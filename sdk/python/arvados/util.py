# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados utilities

This module provides functions and constants that are useful across a variety
of Arvados resource types, or extend the Arvados API client (see `arvados.api`).
"""

import errno
import fcntl
import functools
import hashlib
import httplib2
import os
import random
import re
import subprocess
import sys
import warnings

import arvados.errors

from typing import (
    Any,
    Callable,
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
uuid_pattern = re.compile(r'[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}')
"""Regular expression to match any Arvados object UUID"""
collection_uuid_pattern = re.compile(r'[a-z0-9]{5}-4zz18-[a-z0-9]{15}')
"""Regular expression to match any Arvados collection UUID"""
group_uuid_pattern = re.compile(r'[a-z0-9]{5}-j7d0g-[a-z0-9]{15}')
"""Regular expression to match any Arvados group UUID"""
user_uuid_pattern = re.compile(r'[a-z0-9]{5}-tpzed-[a-z0-9]{15}')
"""Regular expression to match any Arvados user UUID"""
link_uuid_pattern = re.compile(r'[a-z0-9]{5}-o0j2j-[a-z0-9]{15}')
"""Regular expression to match any Arvados link UUID"""
job_uuid_pattern = re.compile(r'[a-z0-9]{5}-8i9sb-[a-z0-9]{15}')
"""Regular expression to match any Arvados job UUID

.. WARNING:: Deprecated
   Arvados job resources are deprecated and will be removed in a future
   release. Prefer the containers API instead.
"""
container_uuid_pattern = re.compile(r'[a-z0-9]{5}-dz642-[a-z0-9]{15}')
"""Regular expression to match any Arvados container UUID"""
manifest_pattern = re.compile(r'((\S+)( +[a-f0-9]{32}(\+[0-9]+)(\+\S+)*)+( +[0-9]+:[0-9]+:\S+)+$)+', flags=re.MULTILINE)
"""Regular expression to match an Arvados collection manifest text"""
keep_file_locator_pattern = re.compile(r'([0-9a-f]{32}\+[0-9]+)/(.*)')
"""Regular expression to match a file path from a collection identified by portable data hash"""
keepuri_pattern = re.compile(r'keep:([0-9a-f]{32}\+[0-9]+)/(.*)')
"""Regular expression to match a `keep:` URI with a collection identified by portable data hash"""

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

    try:
        select = set(kwargs['select'])
    except KeyError:
        pass
    else:
        select.add(order_key)
        select.add('uuid')
        kwargs['select'] = list(select)

    nextpage = []
    tot = 0
    expect_full_page = True
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
            if i["uuid"] in seen_prevpage:
                continue
            seen_thispage.add(i["uuid"])
            yield i

        firstitem = items["items"][0]
        lastitem = items["items"][-1]

        if firstitem[order_key] == lastitem[order_key]:
            # Got a page where every item has the same order key.
            # Switch to using uuid for paging.
            nextpage = [[order_key, "=", lastitem[order_key]], ["uuid", ">" if ascending else "<", lastitem["uuid"]]]
            prev_page_all_same_order_key = True
        else:
            # Start from the last order key seen, but skip the last
            # known uuid to avoid retrieving the same row twice.  If
            # there are multiple rows with the same order key it is
            # still likely we'll end up retrieving duplicate rows.
            # That's handled by tracking the "seen" rows for each page
            # so they can be skipped if they show up on the next page.
            nextpage = [[order_key, ">=" if ascending else "<=", lastitem[order_key]], ["uuid", "!=", lastitem["uuid"]]]
            prev_page_all_same_order_key = False

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

@_deprecated('3.0', 'arvados.util.keyset_list_all')
def list_all(fn, num_retries=0, **kwargs):
    # Default limit to (effectively) api server's MAX_LIMIT
    kwargs.setdefault('limit', sys.maxsize)
    items = []
    offset = 0
    items_available = sys.maxsize
    while len(items) < items_available:
        c = fn(offset=offset, **kwargs).execute(num_retries=num_retries)
        items += c['items']
        items_available = c['items_available']
        offset = c['offset'] + len(c['items'])
    return items

@_deprecated('3.0')
def clear_tmpdir(path=None):
    """
    Ensure the given directory (or TASK_TMPDIR if none given)
    exists and is empty.
    """
    from arvados import current_task
    if path is None:
        path = current_task().tmpdir
    if os.path.exists(path):
        p = subprocess.Popen(['rm', '-rf', path])
        stdout, stderr = p.communicate(None)
        if p.returncode != 0:
            raise Exception('rm -rf %s: %s' % (path, stderr))
    os.mkdir(path)

@_deprecated('3.0', 'subprocess.run')
def run_command(execargs, **kwargs):
    kwargs.setdefault('stdin', subprocess.PIPE)
    kwargs.setdefault('stdout', subprocess.PIPE)
    kwargs.setdefault('stderr', sys.stderr)
    kwargs.setdefault('close_fds', True)
    kwargs.setdefault('shell', False)
    p = subprocess.Popen(execargs, **kwargs)
    stdoutdata, stderrdata = p.communicate(None)
    if p.returncode != 0:
        raise arvados.errors.CommandFailedError(
            "run_command %s exit %d:\n%s" %
            (execargs, p.returncode, stderrdata))
    return stdoutdata, stderrdata

@_deprecated('3.0')
def git_checkout(url, version, path):
    from arvados import current_job
    if not re.search('^/', path):
        path = os.path.join(current_job().tmpdir, path)
    if not os.path.exists(path):
        run_command(["git", "clone", url, path],
                    cwd=os.path.dirname(path))
    run_command(["git", "checkout", version],
                cwd=path)
    return path

@_deprecated('3.0')
def tar_extractor(path, decompress_flag):
    return subprocess.Popen(["tar",
                             "-C", path,
                             ("-x%sf" % decompress_flag),
                             "-"],
                            stdout=None,
                            stdin=subprocess.PIPE, stderr=sys.stderr,
                            shell=False, close_fds=True)

@_deprecated('3.0', 'arvados.collection.Collection.open and the tarfile module')
def tarball_extract(tarball, path):
    """Retrieve a tarball from Keep and extract it to a local
    directory.  Return the absolute path where the tarball was
    extracted. If the top level of the tarball contained just one
    file or directory, return the absolute path of that single
    item.

    tarball -- collection locator
    path -- where to extract the tarball: absolute, or relative to job tmp
    """
    from arvados import current_job
    from arvados.collection import CollectionReader
    if not re.search('^/', path):
        path = os.path.join(current_job().tmpdir, path)
    lockfile = open(path + '.lock', 'w')
    fcntl.flock(lockfile, fcntl.LOCK_EX)
    try:
        os.stat(path)
    except OSError:
        os.mkdir(path)
    already_have_it = False
    try:
        if os.readlink(os.path.join(path, '.locator')) == tarball:
            already_have_it = True
    except OSError:
        pass
    if not already_have_it:

        # emulate "rm -f" (i.e., if the file does not exist, we win)
        try:
            os.unlink(os.path.join(path, '.locator'))
        except OSError:
            if os.path.exists(os.path.join(path, '.locator')):
                os.unlink(os.path.join(path, '.locator'))

        for f in CollectionReader(tarball).all_files():
            f_name = f.name()
            if f_name.endswith(('.tbz', '.tar.bz2')):
                p = tar_extractor(path, 'j')
            elif f_name.endswith(('.tgz', '.tar.gz')):
                p = tar_extractor(path, 'z')
            elif f_name.endswith('.tar'):
                p = tar_extractor(path, '')
            else:
                raise arvados.errors.AssertionError(
                    "tarball_extract cannot handle filename %s" % f.name())
            while True:
                buf = f.read(2**20)
                if len(buf) == 0:
                    break
                p.stdin.write(buf)
            p.stdin.close()
            p.wait()
            if p.returncode != 0:
                lockfile.close()
                raise arvados.errors.CommandFailedError(
                    "tar exited %d" % p.returncode)
        os.symlink(tarball, os.path.join(path, '.locator'))
    tld_extracts = [f for f in os.listdir(path) if f != '.locator']
    lockfile.close()
    if len(tld_extracts) == 1:
        return os.path.join(path, tld_extracts[0])
    return path

@_deprecated('3.0', 'arvados.collection.Collection.open and the zipfile module')
def zipball_extract(zipball, path):
    """Retrieve a zip archive from Keep and extract it to a local
    directory.  Return the absolute path where the archive was
    extracted. If the top level of the archive contained just one
    file or directory, return the absolute path of that single
    item.

    zipball -- collection locator
    path -- where to extract the archive: absolute, or relative to job tmp
    """
    from arvados import current_job
    from arvados.collection import CollectionReader
    if not re.search('^/', path):
        path = os.path.join(current_job().tmpdir, path)
    lockfile = open(path + '.lock', 'w')
    fcntl.flock(lockfile, fcntl.LOCK_EX)
    try:
        os.stat(path)
    except OSError:
        os.mkdir(path)
    already_have_it = False
    try:
        if os.readlink(os.path.join(path, '.locator')) == zipball:
            already_have_it = True
    except OSError:
        pass
    if not already_have_it:

        # emulate "rm -f" (i.e., if the file does not exist, we win)
        try:
            os.unlink(os.path.join(path, '.locator'))
        except OSError:
            if os.path.exists(os.path.join(path, '.locator')):
                os.unlink(os.path.join(path, '.locator'))

        for f in CollectionReader(zipball).all_files():
            if not f.name().endswith('.zip'):
                raise arvados.errors.NotImplementedError(
                    "zipball_extract cannot handle filename %s" % f.name())
            zip_filename = os.path.join(path, os.path.basename(f.name()))
            zip_file = open(zip_filename, 'wb')
            while True:
                buf = f.read(2**20)
                if len(buf) == 0:
                    break
                zip_file.write(buf)
            zip_file.close()

            p = subprocess.Popen(["unzip",
                                  "-q", "-o",
                                  "-d", path,
                                  zip_filename],
                                 stdout=None,
                                 stdin=None, stderr=sys.stderr,
                                 shell=False, close_fds=True)
            p.wait()
            if p.returncode != 0:
                lockfile.close()
                raise arvados.errors.CommandFailedError(
                    "unzip exited %d" % p.returncode)
            os.unlink(zip_filename)
        os.symlink(zipball, os.path.join(path, '.locator'))
    tld_extracts = [f for f in os.listdir(path) if f != '.locator']
    lockfile.close()
    if len(tld_extracts) == 1:
        return os.path.join(path, tld_extracts[0])
    return path

@_deprecated('3.0', 'arvados.collection.Collection')
def collection_extract(collection, path, files=[], decompress=True):
    """Retrieve a collection from Keep and extract it to a local
    directory.  Return the absolute path where the collection was
    extracted.

    collection -- collection locator
    path -- where to extract: absolute, or relative to job tmp
    """
    from arvados import current_job
    from arvados.collection import CollectionReader
    matches = re.search(r'^([0-9a-f]+)(\+[\w@]+)*$', collection)
    if matches:
        collection_hash = matches.group(1)
    else:
        collection_hash = hashlib.md5(collection).hexdigest()
    if not re.search('^/', path):
        path = os.path.join(current_job().tmpdir, path)
    lockfile = open(path + '.lock', 'w')
    fcntl.flock(lockfile, fcntl.LOCK_EX)
    try:
        os.stat(path)
    except OSError:
        os.mkdir(path)
    already_have_it = False
    try:
        if os.readlink(os.path.join(path, '.locator')) == collection_hash:
            already_have_it = True
    except OSError:
        pass

    # emulate "rm -f" (i.e., if the file does not exist, we win)
    try:
        os.unlink(os.path.join(path, '.locator'))
    except OSError:
        if os.path.exists(os.path.join(path, '.locator')):
            os.unlink(os.path.join(path, '.locator'))

    files_got = []
    for s in CollectionReader(collection).all_streams():
        stream_name = s.name()
        for f in s.all_files():
            if (files == [] or
                ((f.name() not in files_got) and
                 (f.name() in files or
                  (decompress and f.decompressed_name() in files)))):
                outname = f.decompressed_name() if decompress else f.name()
                files_got += [outname]
                if os.path.exists(os.path.join(path, stream_name, outname)):
                    continue
                mkdir_dash_p(os.path.dirname(os.path.join(path, stream_name, outname)))
                outfile = open(os.path.join(path, stream_name, outname), 'wb')
                for buf in (f.readall_decompressed() if decompress
                            else f.readall()):
                    outfile.write(buf)
                outfile.close()
    if len(files_got) < len(files):
        raise arvados.errors.AssertionError(
            "Wanted files %s but only got %s from %s" %
            (files, files_got,
             [z.name() for z in CollectionReader(collection).all_files()]))
    os.symlink(collection_hash, os.path.join(path, '.locator'))

    lockfile.close()
    return path

@_deprecated('3.0', 'pathlib.Path().mkdir(parents=True, exist_ok=True)')
def mkdir_dash_p(path):
    if not os.path.isdir(path):
        try:
            os.makedirs(path)
        except OSError as e:
            if e.errno == errno.EEXIST and os.path.isdir(path):
                # It is not an error if someone else creates the
                # directory between our exists() and makedirs() calls.
                pass
            else:
                raise

@_deprecated('3.0', 'arvados.collection.Collection')
def stream_extract(stream, path, files=[], decompress=True):
    """Retrieve a stream from Keep and extract it to a local
    directory.  Return the absolute path where the stream was
    extracted.

    stream -- StreamReader object
    path -- where to extract: absolute, or relative to job tmp
    """
    from arvados import current_job
    if not re.search('^/', path):
        path = os.path.join(current_job().tmpdir, path)
    lockfile = open(path + '.lock', 'w')
    fcntl.flock(lockfile, fcntl.LOCK_EX)
    try:
        os.stat(path)
    except OSError:
        os.mkdir(path)

    files_got = []
    for f in stream.all_files():
        if (files == [] or
            ((f.name() not in files_got) and
             (f.name() in files or
              (decompress and f.decompressed_name() in files)))):
            outname = f.decompressed_name() if decompress else f.name()
            files_got += [outname]
            if os.path.exists(os.path.join(path, outname)):
                os.unlink(os.path.join(path, outname))
            mkdir_dash_p(os.path.dirname(os.path.join(path, outname)))
            outfile = open(os.path.join(path, outname), 'wb')
            for buf in (f.readall_decompressed() if decompress
                        else f.readall()):
                outfile.write(buf)
            outfile.close()
    if len(files_got) < len(files):
        raise arvados.errors.AssertionError(
            "Wanted files %s but only got %s from %s" %
            (files, files_got, [z.name() for z in stream.all_files()]))
    lockfile.close()
    return path

@_deprecated('3.0', 'os.walk')
def listdir_recursive(dirname, base=None, max_depth=None):
    """listdir_recursive(dirname, base, max_depth)

    Return a list of file and directory names found under dirname.

    If base is not None, prepend "{base}/" to each returned name.

    If max_depth is None, descend into directories and return only the
    names of files found in the directory tree.

    If max_depth is a non-negative integer, stop descending into
    directories at the given depth, and at that point return directory
    names instead.

    If max_depth==0 (and base is None) this is equivalent to
    sorted(os.listdir(dirname)).
    """
    allfiles = []
    for ent in sorted(os.listdir(dirname)):
        ent_path = os.path.join(dirname, ent)
        ent_base = os.path.join(base, ent) if base else ent
        if os.path.isdir(ent_path) and max_depth != 0:
            allfiles += listdir_recursive(
                ent_path, base=ent_base,
                max_depth=(max_depth-1 if max_depth else None))
        else:
            allfiles += [ent_base]
    return allfiles
