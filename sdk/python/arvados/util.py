# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import division
from builtins import range

import fcntl
import hashlib
import httplib2
import os
import random
import re
import subprocess
import errno
import sys

import arvados
from arvados.collection import CollectionReader

HEX_RE = re.compile(r'^[0-9a-fA-F]+$')
CR_UNCOMMITTED = 'Uncommitted'
CR_COMMITTED = 'Committed'
CR_FINAL = 'Final'

keep_locator_pattern = re.compile(r'[0-9a-f]{32}\+\d+(\+\S+)*')
signed_locator_pattern = re.compile(r'[0-9a-f]{32}\+\d+(\+\S+)*\+A\S+(\+\S+)*')
portable_data_hash_pattern = re.compile(r'[0-9a-f]{32}\+\d+')
uuid_pattern = re.compile(r'[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}')
collection_uuid_pattern = re.compile(r'[a-z0-9]{5}-4zz18-[a-z0-9]{15}')
group_uuid_pattern = re.compile(r'[a-z0-9]{5}-j7d0g-[a-z0-9]{15}')
user_uuid_pattern = re.compile(r'[a-z0-9]{5}-tpzed-[a-z0-9]{15}')
link_uuid_pattern = re.compile(r'[a-z0-9]{5}-o0j2j-[a-z0-9]{15}')
job_uuid_pattern = re.compile(r'[a-z0-9]{5}-8i9sb-[a-z0-9]{15}')
container_uuid_pattern = re.compile(r'[a-z0-9]{5}-dz642-[a-z0-9]{15}')
manifest_pattern = re.compile(r'((\S+)( +[a-f0-9]{32}(\+\d+)(\+\S+)*)+( +\d+:\d+:\S+)+$)+', flags=re.MULTILINE)

def clear_tmpdir(path=None):
    """
    Ensure the given directory (or TASK_TMPDIR if none given)
    exists and is empty.
    """
    if path is None:
        path = arvados.current_task().tmpdir
    if os.path.exists(path):
        p = subprocess.Popen(['rm', '-rf', path])
        stdout, stderr = p.communicate(None)
        if p.returncode != 0:
            raise Exception('rm -rf %s: %s' % (path, stderr))
    os.mkdir(path)

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

def git_checkout(url, version, path):
    if not re.search('^/', path):
        path = os.path.join(arvados.current_job().tmpdir, path)
    if not os.path.exists(path):
        run_command(["git", "clone", url, path],
                    cwd=os.path.dirname(path))
    run_command(["git", "checkout", version],
                cwd=path)
    return path

def tar_extractor(path, decompress_flag):
    return subprocess.Popen(["tar",
                             "-C", path,
                             ("-x%sf" % decompress_flag),
                             "-"],
                            stdout=None,
                            stdin=subprocess.PIPE, stderr=sys.stderr,
                            shell=False, close_fds=True)

def tarball_extract(tarball, path):
    """Retrieve a tarball from Keep and extract it to a local
    directory.  Return the absolute path where the tarball was
    extracted. If the top level of the tarball contained just one
    file or directory, return the absolute path of that single
    item.

    tarball -- collection locator
    path -- where to extract the tarball: absolute, or relative to job tmp
    """
    if not re.search('^/', path):
        path = os.path.join(arvados.current_job().tmpdir, path)
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
            if re.search('\.(tbz|tar.bz2)$', f.name()):
                p = tar_extractor(path, 'j')
            elif re.search('\.(tgz|tar.gz)$', f.name()):
                p = tar_extractor(path, 'z')
            elif re.search('\.tar$', f.name()):
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

def zipball_extract(zipball, path):
    """Retrieve a zip archive from Keep and extract it to a local
    directory.  Return the absolute path where the archive was
    extracted. If the top level of the archive contained just one
    file or directory, return the absolute path of that single
    item.

    zipball -- collection locator
    path -- where to extract the archive: absolute, or relative to job tmp
    """
    if not re.search('^/', path):
        path = os.path.join(arvados.current_job().tmpdir, path)
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
            if not re.search('\.zip$', f.name()):
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

def collection_extract(collection, path, files=[], decompress=True):
    """Retrieve a collection from Keep and extract it to a local
    directory.  Return the absolute path where the collection was
    extracted.

    collection -- collection locator
    path -- where to extract: absolute, or relative to job tmp
    """
    matches = re.search(r'^([0-9a-f]+)(\+[\w@]+)*$', collection)
    if matches:
        collection_hash = matches.group(1)
    else:
        collection_hash = hashlib.md5(collection).hexdigest()
    if not re.search('^/', path):
        path = os.path.join(arvados.current_job().tmpdir, path)
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

def stream_extract(stream, path, files=[], decompress=True):
    """Retrieve a stream from Keep and extract it to a local
    directory.  Return the absolute path where the stream was
    extracted.

    stream -- StreamReader object
    path -- where to extract: absolute, or relative to job tmp
    """
    if not re.search('^/', path):
        path = os.path.join(arvados.current_job().tmpdir, path)
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

def is_hex(s, *length_args):
    """is_hex(s[, length[, max_length]]) -> boolean

    Return True if s is a string of hexadecimal digits.
    If one length argument is given, the string must contain exactly
    that number of digits.
    If two length arguments are given, the string must contain a number of
    digits between those two lengths, inclusive.
    Return False otherwise.
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

def keyset_list_all(fn, order_key="created_at", num_retries=0, ascending=True, **kwargs):
    pagesize = 1000
    kwargs["limit"] = pagesize
    kwargs["count"] = 'none'
    asc = "asc" if ascending else "desc"
    kwargs["order"] = ["%s %s" % (order_key, asc), "uuid %s" % asc]
    other_filters = kwargs.get("filters", [])

    if "select" in kwargs and "uuid" not in kwargs["select"]:
        kwargs["select"].append("uuid")

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


def ca_certs_path(fallback=httplib2.CA_CERTS):
    """Return the path of the best available CA certs source.

    This function searches for various distribution sources of CA
    certificates, and returns the first it finds.  If it doesn't find any,
    it returns the value of `fallback` (httplib2's CA certs by default).
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

def new_request_id():
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

def get_config_once(svc):
    if not svc._rootDesc.get('resources').get('configs', False):
        # Old API server version, no config export endpoint
        return {}
    if not hasattr(svc, '_cached_config'):
        svc._cached_config = svc.configs().get().execute()
    return svc._cached_config

def get_vocabulary_once(svc):
    if not svc._rootDesc.get('resources').get('vocabularies', False):
        # Old API server version, no vocabulary export endpoint
        return {}
    if not hasattr(svc, '_cached_vocabulary'):
        svc._cached_vocabulary = svc.vocabularies().get().execute()
    return svc._cached_vocabulary
