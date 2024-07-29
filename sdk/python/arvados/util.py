# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados utilities

This module provides functions and constants that are useful across a variety
of Arvados resource types, or extend the Arvados API client (see `arvados.api`).
"""

import errno
import fcntl
import hashlib
import httplib2
import operator
import os
import random
import re
import subprocess
import sys

import arvados.errors

from typing import (
    Any,
    Callable,
    Container,
    Dict,
    Iterator,
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
        key_fields: Container[str]=('uuid',),
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

    * key_fields: Container[str] --- One or two fields that constitute
      a unique key for returned items.  Normally this should be the
      default value `('uuid',)`, unless `fn` returns
      computed_permissions records, in which case it should be
      `('user_uuid', 'target_uuid')`.  If two fields are given, one of
      them must be equal to `order_key`.

    Additional keyword arguments will be passed directly to `fn` for each API
    call. Note that this function sets `count`, `limit`, and `order` as part of
    its work.

    """
    tiebreak_keys = set(key_fields) - {order_key}
    if len(tiebreak_keys) == 0:
        tiebreak_key = 'uuid'
    elif len(tiebreak_keys) == 1:
        tiebreak_key = tiebreak_keys.pop()
    else:
        raise arvados.errors.ArgumentError(
            "key_fields can have at most one entry that is not order_key")

    pagesize = 1000
    kwargs["limit"] = pagesize
    kwargs["count"] = 'none'
    asc = "asc" if ascending else "desc"
    kwargs["order"] = [f"{order_key} {asc}", f"{tiebreak_key} {asc}"]
    other_filters = kwargs.get("filters", [])

    if 'select' in kwargs:
        kwargs['select'] = list({*kwargs['select'], *key_fields, order_key})

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
        key_fields: Container[str]=('user_uuid', 'target_uuid'),
        **kwargs: Any,
) -> Iterator[Dict[str, Any]]:
    """Iterate all `computed_permission` resources

    This method is the same as `keyset_list_all`, except that its
    default arguments are suitable for the computed_permissions API.

    Arguments:

    * fn: Callable[..., arvados.api_resources.ArvadosAPIRequest] ---
      see `keyset_list_all`.  Typically this is an instance of
      `arvados.api_resources.ComputedPermissions.list`.  Given an
      Arvados API client named `arv`, typical usage is
      `iter_computed_permissions(arv.computed_permissions().list)`.

    * order_key: str --- see `keyset_list_all`.  Default
      `'user_uuid'`.

    * num_retries: int --- see `keyset_list_all`.

    * ascending: bool --- see `keyset_list_all`.

    * key_fields: Container[str] --- see `keyset_list_all`. Default
      `('user_uuid', 'target_uuid')`.

    """
    return keyset_list_all(
        fn=fn,
        order_key=order_key,
        num_retries=num_retries,
        ascending=ascending,
        key_fields=key_fields,
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
