# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados utilities

This module provides functions and constants that are useful across a variety
of Arvados resource types, or extend the Arvados API client (see `arvados.api`).
"""

import dataclasses
import enum
import errno
import fcntl
import functools
import hashlib
import httplib2
import itertools
import logging
import operator
import os
import random
import re
import shlex
import stat
import subprocess
import sys
import warnings

import arvados.errors

from pathlib import Path, PurePath
from typing import (
    Any,
    Callable,
    Dict,
    Iterator,
    Mapping,
    Optional,
    Sequence,
    TypeVar,
    Union,
)

T = TypeVar('T')

HEX_RE = re.compile(r'^[0-9a-fA-F]+$')
"""Regular expression to match a hexadecimal string (case-insensitive)"""
CR_UNCOMMITTED = 'Uncommitted'
"""Constant `state` value for uncommited container requests"""
CR_COMMITTED = 'Committed'
"""Constant `state` value for committed container requests"""
CR_FINAL = 'Final'
"""Constant `state` value for finalized container requests"""

keep_locator_pattern = re.compile(r'[0-9a-f]{32}\+[0-9]+(\+\S+)*')
"""Regular expression to match any Keep block locator"""
signed_locator_pattern = re.compile(r'[0-9a-f]{32}\+[0-9]+(\+\S+)*\+A\S+(\+\S+)*')
"""Regular expression to match any Keep block locator with an access token hint"""
portable_data_hash_pattern = re.compile(r'[0-9a-f]{32}\+[0-9]+')
"""Regular expression to match any collection portable data hash"""
manifest_pattern = re.compile(r'((\S+)( +[a-f0-9]{32}(\+[0-9]+)(\+\S+)*)+( +[0-9]+:[0-9]+:\S+)+$)+', flags=re.MULTILINE)
"""Regular expression to match an Arvados collection manifest text"""
keep_file_locator_pattern = re.compile(r'([0-9a-f]{32}\+[0-9]+)/(.*)')
"""Regular expression to match a file path from a collection identified by portable data hash"""
keepuri_pattern = re.compile(r'keep:([0-9a-f]{32}\+[0-9]+)/(.*)')
"""Regular expression to match a `keep:` URI with a collection identified by portable data hash"""

uuid_pattern = re.compile(r'[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}')
"""Regular expression to match any Arvados object UUID"""
collection_uuid_pattern = re.compile(r'[a-z0-9]{5}-4zz18-[a-z0-9]{15}')
"""Regular expression to match any Arvados collection UUID"""
container_uuid_pattern = re.compile(r'[a-z0-9]{5}-dz642-[a-z0-9]{15}')
"""Regular expression to match any Arvados container UUID"""
group_uuid_pattern = re.compile(r'[a-z0-9]{5}-j7d0g-[a-z0-9]{15}')
"""Regular expression to match any Arvados group UUID"""
link_uuid_pattern = re.compile(r'[a-z0-9]{5}-o0j2j-[a-z0-9]{15}')
"""Regular expression to match any Arvados link UUID"""
user_uuid_pattern = re.compile(r'[a-z0-9]{5}-tpzed-[a-z0-9]{15}')
"""Regular expression to match any Arvados user UUID"""

logger = logging.getLogger('arvados')

def _deprecated(version=None, preferred=None):
    """Mark a callable as deprecated in the SDK

    This will wrap the callable to emit as a DeprecationWarning
    and add a deprecation notice to its docstring.

    If the following arguments are given, they'll be included in the
    notices:

    * preferred: str | None --- The name of an alternative that users should
      use instead.

    * version: str | None --- The version of Arvados when the callable is
      scheduled to be removed.
    """
    if version is None:
        version = ''
    else:
        version = f' and scheduled to be removed in Arvados {version}'
    if preferred is None:
        preferred = ''
    else:
        preferred = f' Prefer {preferred} instead.'
    def deprecated_decorator(func):
        fullname = f'{func.__module__}.{func.__qualname__}'
        parent, _, name = fullname.rpartition('.')
        if name == '__init__':
            fullname = parent
        warning_msg = f'{fullname} is deprecated{version}.{preferred}'
        @functools.wraps(func)
        def deprecated_wrapper(*args, **kwargs):
            warnings.warn(warning_msg, DeprecationWarning, 2)
            return func(*args, **kwargs)
        # Get func's docstring without any trailing newline or empty lines.
        func_doc = re.sub(r'\n\s*$', '', func.__doc__ or '')
        match = re.search(r'\n([ \t]+)\S', func_doc)
        indent = '' if match is None else match.group(1)
        warning_doc = f'\n\n{indent}.. WARNING:: Deprecated\n{indent}   {warning_msg}'
        # Make the deprecation notice the second "paragraph" of the
        # docstring if possible. Otherwise append it.
        docstring, count = re.subn(
            rf'\n[ \t]*\n{indent}',
            f'{warning_doc}\n\n{indent}',
            func_doc,
            count=1,
        )
        if not count:
            docstring = f'{func_doc.lstrip()}{warning_doc}'
        deprecated_wrapper.__doc__ = docstring
        return deprecated_wrapper
    return deprecated_decorator

@dataclasses.dataclass
class _BaseDirectorySpec:
    """Parse base directories

    A _BaseDirectorySpec defines all the environment variable keys and defaults
    related to a set of base directories (cache, config, state, etc.). It
    provides pure methods to parse environment settings into valid paths.
    """
    systemd_key: str
    xdg_home_key: str
    xdg_home_default: PurePath
    xdg_dirs_key: Optional[str] = None
    xdg_dirs_default: str = ''

    @staticmethod
    def _abspath_from_env(env: Mapping[str, str], key: str) -> Optional[Path]:
        try:
            path = Path(env[key])
        except (KeyError, ValueError):
            ok = False
        else:
            ok = path.is_absolute()
        return path if ok else None

    @staticmethod
    def _iter_abspaths(value: str) -> Iterator[Path]:
        for path_s in value.split(':'):
            path = Path(path_s)
            if path.is_absolute():
                yield path

    def iter_systemd(self, env: Mapping[str, str]) -> Iterator[Path]:
        return self._iter_abspaths(env.get(self.systemd_key, ''))

    def iter_xdg(self, env: Mapping[str, str], subdir: PurePath) -> Iterator[Path]:
        yield self.xdg_home(env, subdir)
        if self.xdg_dirs_key is not None:
            for path in self._iter_abspaths(env.get(self.xdg_dirs_key) or self.xdg_dirs_default):
                yield path / subdir

    def xdg_home(self, env: Mapping[str, str], subdir: PurePath) -> Path:
        return (
            self._abspath_from_env(env, self.xdg_home_key)
            or self.xdg_home_default_path(env)
        ) / subdir

    def xdg_home_default_path(self, env: Mapping[str, str]) -> Path:
        return (self._abspath_from_env(env, 'HOME') or Path.home()) / self.xdg_home_default

    def xdg_home_is_customized(self, env: Mapping[str, str]) -> bool:
        xdg_home = self._abspath_from_env(env, self.xdg_home_key)
        return xdg_home is not None and xdg_home != self.xdg_home_default_path(env)


class _BaseDirectorySpecs(enum.Enum):
    """Base directory specifications

    This enum provides easy access to the standard base directory settings.
    """
    CACHE = _BaseDirectorySpec(
        'CACHE_DIRECTORY',
        'XDG_CACHE_HOME',
        PurePath('.cache'),
    )
    CONFIG = _BaseDirectorySpec(
        'CONFIGURATION_DIRECTORY',
        'XDG_CONFIG_HOME',
        PurePath('.config'),
        'XDG_CONFIG_DIRS',
        '/etc/xdg',
    )
    STATE = _BaseDirectorySpec(
        'STATE_DIRECTORY',
        'XDG_STATE_HOME',
        PurePath('.local', 'state'),
    )


class _BaseDirectories:
    """Resolve paths from a base directory spec

    Given a _BaseDirectorySpec, this class provides stateful methods to find
    existing files and return the most-preferred directory for writing.
    """
    _STORE_MODE = stat.S_IFDIR | stat.S_IWUSR

    def __init__(
            self,
            spec: Union[_BaseDirectorySpec, _BaseDirectorySpecs, str],
            env: Mapping[str, str]=os.environ,
            xdg_subdir: Union[os.PathLike, str]='arvados',
    ) -> None:
        if isinstance(spec, str):
            spec = _BaseDirectorySpecs[spec].value
        elif isinstance(spec, _BaseDirectorySpecs):
            spec = spec.value
        self._spec = spec
        self._env = env
        self._xdg_subdir = PurePath(xdg_subdir)

    def search(self, name: str) -> Iterator[Path]:
        any_found = False
        for search_path in itertools.chain(
                self._spec.iter_systemd(self._env),
                self._spec.iter_xdg(self._env, self._xdg_subdir),
        ):
            path = search_path / name
            if path.exists():
                yield path
                any_found = True
        # The rest of this function is dedicated to warning the user if they
        # have a custom XDG_*_HOME value that prevented the search from
        # succeeding. This should be rare.
        if any_found or not self._spec.xdg_home_is_customized(self._env):
            return
        default_home = self._spec.xdg_home_default_path(self._env)
        default_path = Path(self._xdg_subdir / name)
        if not (default_home / default_path).exists():
            return
        if self._spec.xdg_dirs_key is None:
            suggest_key = self._spec.xdg_home_key
            suggest_value = default_home
        else:
            suggest_key = self._spec.xdg_dirs_key
            cur_value = self._env.get(suggest_key, '')
            value_sep = ':' if cur_value else ''
            suggest_value = f'{cur_value}{value_sep}{default_home}'
        logger.warning(
            "\
%s was not found under your configured $%s (%s), \
but does exist at the default location (%s) - \
consider running this program with the environment setting %s=%s\
",
            default_path,
            self._spec.xdg_home_key,
            self._spec.xdg_home(self._env, ''),
            default_home,
            suggest_key,
            shlex.quote(suggest_value),
        )

    def storage_path(
            self,
            subdir: Union[str, os.PathLike]=PurePath(),
            mode: int=0o700,
    ) -> Path:
        for path in self._spec.iter_systemd(self._env):
            try:
                mode = path.stat().st_mode
            except OSError:
                continue
            if (mode & self._STORE_MODE) == self._STORE_MODE:
                break
        else:
            path = self._spec.xdg_home(self._env, self._xdg_subdir)
        path /= subdir
        path.mkdir(parents=True, exist_ok=True, mode=mode)
        return path


def is_hex(s: str, *length_args: int) -> bool:
    """Indicate whether a string is a hexadecimal number

    This method returns true if all characters in the string are hexadecimal
    digits. It is case-insensitive.

    You can also pass optional length arguments to check that the string has
    the expected number of digits. If you pass one integer, the string must
    have that length exactly, otherwise the method returns False. If you
    pass two integers, the string's length must fall within that minimum and
    maximum (inclusive), otherwise the method returns False.

    Arguments:

    * s: str --- The string to check

    * length_args: int --- Optional length limit(s) for the string to check
    """
    num_length_args = len(length_args)
    if num_length_args > 2:
        raise arvados.errors.ArgumentError(
            "is_hex accepts up to 3 arguments ({} given)".format(1 + num_length_args))
    elif num_length_args == 2:
        good_len = (length_args[0] <= len(s) <= length_args[1])
    elif num_length_args == 1:
        good_len = (len(s) == length_args[0])
    else:
        good_len = True
    return bool(good_len and HEX_RE.match(s))

def keyset_list_all(
        fn: Callable[..., 'arvados.api_resources.ArvadosAPIRequest'],
        order_key: str="created_at",
        num_retries: int=0,
        ascending: bool=True,
        key_fields: Sequence[str]=('uuid',),
        **kwargs: Any,
) -> Iterator[Dict[str, Any]]:
    """Iterate all Arvados resources from an API list call

    This method takes a method that represents an Arvados API list call, and
    iterates the objects returned by the API server. It can make multiple API
    calls to retrieve and iterate all objects available from the API server.

    Arguments:

    * fn: Callable[..., arvados.api_resources.ArvadosAPIRequest] --- A
      function that wraps an Arvados API method that returns a list of
      objects. If you have an Arvados API client named `arv`, examples
      include `arv.collections().list` and `arv.groups().contents`. Note
      that you should pass the function *without* calling it.

    * order_key: str --- The name of the primary object field that objects
      should be sorted by. This name is used to build an `order` argument
      for `fn`. Default `'created_at'`.

    * num_retries: int --- This argument is passed through to
      `arvados.api_resources.ArvadosAPIRequest.execute` for each API call. See
      that method's docstring for details. Default 0 (meaning API calls will
      use the `num_retries` value set when the Arvados API client was
      constructed).

    * ascending: bool --- Used to build an `order` argument for `fn`. If True,
      all fields will be sorted in `'asc'` (ascending) order. Otherwise, all
      fields will be sorted in `'desc'` (descending) order.

    * key_fields: Sequence[str] --- One or two fields that constitute a
      unique key for returned items.  Normally this should be the
      default value `('uuid',)`, unless `fn` returns
      computed_permissions records, in which case it should be
      `('user_uuid', 'target_uuid')`.  If two fields are given, one of
      them must be equal to `order_key`.

    Additional keyword arguments will be passed directly to `fn` for each API
    call. Note that this function sets `count`, `limit`, and `order` as part of
    its work.

    """
    pagesize = 1000
    kwargs["limit"] = pagesize
    kwargs["count"] = 'none'
    asc = "asc" if ascending else "desc"
    kwargs["order"] = ["%s %s" % (order_key, asc), "uuid %s" % asc]
    other_filters = kwargs.get("filters", [])

    tiebreak_keys = set(key_fields) - {order_key}
    if len(tiebreak_keys) == 0:
        tiebreak_key = 'uuid'
    elif len(tiebreak_keys) == 1:
        tiebreak_key = tiebreak_keys.pop()
    else:
        raise arvados.errors.ArgumentError(
            "key_fields can have at most one entry that is not order_key")

    try:
        select = set(kwargs['select'])
    except KeyError:
        pass
    else:
        kwargs['select'] = list(select | set(key_fields) | {order_key})

    nextpage = []
    tot = 0
    expect_full_page = True
    key_getter = operator.itemgetter(*key_fields)
    seen_prevpage = set()
    seen_thispage = set()
    lastitem = None
    prev_page_all_same_order_key = False

    while True:
        kwargs["filters"] = nextpage+other_filters
        items = fn(**kwargs).execute(num_retries=num_retries)

        if len(items["items"]) == 0:
            if prev_page_all_same_order_key:
                nextpage = [[order_key, ">" if ascending else "<", lastitem[order_key]]]
                prev_page_all_same_order_key = False
                continue
            else:
                return

        seen_prevpage = seen_thispage
        seen_thispage = set()

        for i in items["items"]:
            # In cases where there's more than one record with the
            # same order key, the result could include records we
            # already saw in the last page.  Skip them.
            seen_key = key_getter(i)
            if seen_key in seen_prevpage:
                continue
            seen_thispage.add(seen_key)
            yield i

        firstitem = items["items"][0]
        lastitem = items["items"][-1]

        if firstitem[order_key] == lastitem[order_key]:
            # Got a page where every item has the same order key.
            # Switch to using tiebreak key for paging.
            nextpage = [[order_key, "=", lastitem[order_key]], [tiebreak_key, ">" if ascending else "<", lastitem[tiebreak_key]]]
            prev_page_all_same_order_key = True
        else:
            # Start from the last order key seen, but skip the last
            # known uuid to avoid retrieving the same row twice.  If
            # there are multiple rows with the same order key it is
            # still likely we'll end up retrieving duplicate rows.
            # That's handled by tracking the "seen" rows for each page
            # so they can be skipped if they show up on the next page.
            nextpage = [[order_key, ">=" if ascending else "<=", lastitem[order_key]]]
            if tiebreak_key == "uuid":
                nextpage += [[tiebreak_key, "!=", lastitem[tiebreak_key]]]
            prev_page_all_same_order_key = False

def iter_computed_permissions(
        fn: Callable[..., 'arvados.api_resources.ArvadosAPIRequest'],
        order_key: str='user_uuid',
        num_retries: int=0,
        ascending: bool=True,
        key_fields: Sequence[str]=('user_uuid', 'target_uuid'),
        **kwargs: Any,
) -> Iterator[Dict[str, Any]]:
    """Iterate all `computed_permission` resources

    This method is the same as `keyset_list_all`, except that its
    default arguments are suitable for the computed_permissions API.

    Arguments:

    * fn: Callable[..., arvados.api_resources.ArvadosAPIRequest] ---
      see `keyset_list_all`.  Typically
      `arv.computed_permissions().list`.

    * order_key: str --- see `keyset_list_all`.  Default
      `'user_uuid'`.

    * num_retries: int --- see `keyset_list_all`.

    * ascending: bool --- see `keyset_list_all`.

    * key_fields: Sequence[str] --- see `keyset_list_all`. Default
      `('user_uuid', 'target_uuid')`.

    """
    return keyset_list_all(
        fn,
        order_key='user_uuid',
        key_fields=('user_uuid', 'target_uuid'),
        **kwargs)

def ca_certs_path(fallback: T=httplib2.CA_CERTS) -> Union[str, T]:
    """Return the path of the best available source of CA certificates

    This function checks various known paths that provide trusted CA
    certificates, and returns the first one that exists. It checks:

    * the path in the `SSL_CERT_FILE` environment variable (used by OpenSSL)
    * `/etc/arvados/ca-certificates.crt`, respected by all Arvados software
    * `/etc/ssl/certs/ca-certificates.crt`, the default store on Debian-based
      distributions
    * `/etc/pki/tls/certs/ca-bundle.crt`, the default store on Red Hat-based
      distributions

    If none of these paths exist, this function returns the value of `fallback`.

    Arguments:

    * fallback: T --- The value to return if none of the known paths exist.
      The default value is the certificate store of Mozilla's trusted CAs
      included with the Python [certifi][] package.

    [certifi]: https://pypi.org/project/certifi/
    """
    for ca_certs_path in [
        # SSL_CERT_FILE and SSL_CERT_DIR are openssl overrides - note
        # that httplib2 itself also supports HTTPLIB2_CA_CERTS.
        os.environ.get('SSL_CERT_FILE'),
        # Arvados specific:
        '/etc/arvados/ca-certificates.crt',
        # Debian:
        '/etc/ssl/certs/ca-certificates.crt',
        # Red Hat:
        '/etc/pki/tls/certs/ca-bundle.crt',
        ]:
        if ca_certs_path and os.path.exists(ca_certs_path):
            return ca_certs_path
    return fallback

def new_request_id() -> str:
    """Return a random request ID

    This function generates and returns a random string suitable for use as a
    `X-Request-Id` header value in the Arvados API.
    """
    rid = "req-"
    # 2**104 > 36**20 > 2**103
    n = random.getrandbits(104)
    for _ in range(20):
        c = n % 36
        if c < 10:
            rid += chr(c+ord('0'))
        else:
            rid += chr(c+ord('a')-10)
        n = n // 36
    return rid

def get_config_once(svc: 'arvados.api_resources.ArvadosAPIClient') -> Dict[str, Any]:
    """Return an Arvados cluster's configuration, with caching

    This function gets and returns the Arvados configuration from the API
    server. It caches the result on the client object and reuses it on any
    future calls.

    Arguments:

    * svc: arvados.api_resources.ArvadosAPIClient --- The Arvados API client
      object to use to retrieve and cache the Arvados cluster configuration.
    """
    if not svc._rootDesc.get('resources').get('configs', False):
        # Old API server version, no config export endpoint
        return {}
    if not hasattr(svc, '_cached_config'):
        svc._cached_config = svc.configs().get().execute()
    return svc._cached_config

def get_vocabulary_once(svc: 'arvados.api_resources.ArvadosAPIClient') -> Dict[str, Any]:
    """Return an Arvados cluster's vocabulary, with caching

    This function gets and returns the Arvados vocabulary from the API
    server. It caches the result on the client object and reuses it on any
    future calls.

    .. HINT:: Low-level method
       This is a relatively low-level wrapper around the Arvados API. Most
       users will prefer to use `arvados.vocabulary.load_vocabulary`.

    Arguments:

    * svc: arvados.api_resources.ArvadosAPIClient --- The Arvados API client
      object to use to retrieve and cache the Arvados cluster vocabulary.
    """
    if not svc._rootDesc.get('resources').get('vocabularies', False):
        # Old API server version, no vocabulary export endpoint
        return {}
    if not hasattr(svc, '_cached_vocabulary'):
        svc._cached_vocabulary = svc.vocabularies().get().execute()
    return svc._cached_vocabulary

def trim_name(collectionname: str) -> str:
    """Limit the length of a name to fit within Arvados API limits

    This function ensures that a string is short enough to use as an object
    name in the Arvados API, leaving room for text that may be added by the
    `ensure_unique_name` argument. If the source name is short enough, it is
    returned unchanged. Otherwise, this function returns a string with excess
    characters removed from the middle of the source string and replaced with
    an ellipsis.

    Arguments:

    * collectionname: str --- The desired source name
    """
    max_name_len = 254 - 28

    if len(collectionname) > max_name_len:
        over = len(collectionname) - max_name_len
        split = int(max_name_len/2)
        collectionname = collectionname[0:split] + "…" + collectionname[split+over:]

    return collectionname
