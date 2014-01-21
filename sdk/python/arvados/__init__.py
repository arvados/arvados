import gflags
import httplib
import httplib2
import logging
import os
import pprint
import sys
import types
import subprocess
import json
import UserDict
import re
import hashlib
import string
import bz2
import zlib
import fcntl
import time
import threading

import apiclient
import apiclient.discovery

from stream import *
from collection import *
from keep import *

config = None
EMPTY_BLOCK_LOCATOR = 'd41d8cd98f00b204e9800998ecf8427e+0'
services = {}

# Arvados configuration settings are taken from $HOME/.config/arvados.
# Environment variables override settings in the config file.
#
class ArvadosConfig(dict):
    def __init__(self, config_file):
        dict.__init__(self)
        if os.path.exists(config_file):
            with open(config_file, "r") as f:
                for config_line in f:
                    var, val = config_line.rstrip().split('=', 2)
                    self[var] = val
        for var in os.environ:
            if var.startswith('ARVADOS_'):
                self[var] = os.environ[var]

class errors:
    class SyntaxError(Exception):
        pass
    class AssertionError(Exception):
        pass
    class NotFoundError(Exception):
        pass
    class CommandFailedError(Exception):
        pass
    class KeepWriteError(Exception):
        pass
    class NotImplementedError(Exception):
        pass

class CredentialsFromEnv(object):
    @staticmethod
    def http_request(self, uri, **kwargs):
        global config
        from httplib import BadStatusLine
        if 'headers' not in kwargs:
            kwargs['headers'] = {}
        kwargs['headers']['Authorization'] = 'OAuth2 %s' % config.get('ARVADOS_API_TOKEN', 'ARVADOS_API_TOKEN_not_set')
        try:
            return self.orig_http_request(uri, **kwargs)
        except BadStatusLine:
            # This is how httplib tells us that it tried to reuse an
            # existing connection but it was already closed by the
            # server. In that case, yes, we would like to retry.
            # Unfortunately, we are not absolutely certain that the
            # previous call did not succeed, so this is slightly
            # risky.
            return self.orig_http_request(uri, **kwargs)
    def authorize(self, http):
        http.orig_http_request = http.request
        http.request = types.MethodType(self.http_request, http)
        return http

def task_set_output(self,s):
    api('v1').job_tasks().update(uuid=self['uuid'],
                                 body={
            'output':s,
            'success':True,
            'progress':1.0
            }).execute()

_current_task = None
def current_task():
    global _current_task
    if _current_task:
        return _current_task
    t = api('v1').job_tasks().get(uuid=os.environ['TASK_UUID']).execute()
    t = UserDict.UserDict(t)
    t.set_output = types.MethodType(task_set_output, t)
    t.tmpdir = os.environ['TASK_WORK']
    _current_task = t
    return t

_current_job = None
def current_job():
    global _current_job
    if _current_job:
        return _current_job
    t = api('v1').jobs().get(uuid=os.environ['JOB_UUID']).execute()
    t = UserDict.UserDict(t)
    t.tmpdir = os.environ['JOB_WORK']
    _current_job = t
    return t

def getjobparam(*args):
    return current_job()['script_parameters'].get(*args)

# Monkey patch discovery._cast() so objects and arrays get serialized
# with json.dumps() instead of str().
_cast_orig = apiclient.discovery._cast
def _cast_objects_too(value, schema_type):
    global _cast_orig
    if (type(value) != type('') and
        (schema_type == 'object' or schema_type == 'array')):
        return json.dumps(value)
    else:
        return _cast_orig(value, schema_type)
apiclient.discovery._cast = _cast_objects_too

def api(version=None):
    global services, config

    if not config:
        config = ArvadosConfig(os.environ['HOME'] + '/.config/arvados')
        if 'ARVADOS_DEBUG' in config:
            logging.basicConfig(level=logging.DEBUG)

    if not services.get(version):
        apiVersion = version
        if not version:
            apiVersion = 'v1'
            logging.info("Using default API version. " +
                         "Call arvados.api('%s') instead." %
                         apiVersion)
        if 'ARVADOS_API_HOST' not in config:
            raise Exception("ARVADOS_API_HOST is not set. Aborting.")
        url = ('https://%s/discovery/v1/apis/{api}/{apiVersion}/rest' %
               config['ARVADOS_API_HOST'])
        credentials = CredentialsFromEnv()

        # Use system's CA certificates (if we find them) instead of httplib2's
        ca_certs = '/etc/ssl/certs/ca-certificates.crt'
        if not os.path.exists(ca_certs):
            ca_certs = None             # use httplib2 default

        http = httplib2.Http(ca_certs=ca_certs)
        http = credentials.authorize(http)
        if re.match(r'(?i)^(true|1|yes)$',
                    config.get('ARVADOS_API_HOST_INSECURE', 'no')):
            http.disable_ssl_certificate_validation=True
        services[version] = apiclient.discovery.build(
            'arvados', apiVersion, http=http, discoveryServiceUrl=url)
    return services[version]

class JobTask(object):
    def __init__(self, parameters=dict(), runtime_constraints=dict()):
        print "init jobtask %s %s" % (parameters, runtime_constraints)

class job_setup:
    @staticmethod
    def one_task_per_input_file(if_sequence=0, and_end_task=True):
        if if_sequence != current_task()['sequence']:
            return
        job_input = current_job()['script_parameters']['input']
        cr = CollectionReader(job_input)
        for s in cr.all_streams():
            for f in s.all_files():
                task_input = f.as_manifest()
                new_task_attrs = {
                    'job_uuid': current_job()['uuid'],
                    'created_by_job_task_uuid': current_task()['uuid'],
                    'sequence': if_sequence + 1,
                    'parameters': {
                        'input':task_input
                        }
                    }
                api('v1').job_tasks().create(body=new_task_attrs).execute()
        if and_end_task:
            api('v1').job_tasks().update(uuid=current_task()['uuid'],
                                       body={'success':True}
                                       ).execute()
            exit(0)

    @staticmethod
    def one_task_per_input_stream(if_sequence=0, and_end_task=True):
        if if_sequence != current_task()['sequence']:
            return
        job_input = current_job()['script_parameters']['input']
        cr = CollectionReader(job_input)
        for s in cr.all_streams():
            task_input = s.tokens()
            new_task_attrs = {
                'job_uuid': current_job()['uuid'],
                'created_by_job_task_uuid': current_task()['uuid'],
                'sequence': if_sequence + 1,
                'parameters': {
                    'input':task_input
                    }
                }
            api('v1').job_tasks().create(body=new_task_attrs).execute()
        if and_end_task:
            api('v1').job_tasks().update(uuid=current_task()['uuid'],
                                       body={'success':True}
                                       ).execute()
            exit(0)

class util:
    @staticmethod
    def clear_tmpdir(path=None):
        """
        Ensure the given directory (or TASK_TMPDIR if none given)
        exists and is empty.
        """
        if path == None:
            path = current_task().tmpdir
        if os.path.exists(path):
            p = subprocess.Popen(['rm', '-rf', path])
            stdout, stderr = p.communicate(None)
            if p.returncode != 0:
                raise Exception('rm -rf %s: %s' % (path, stderr))
        os.mkdir(path)

    @staticmethod
    def run_command(execargs, **kwargs):
        kwargs.setdefault('stdin', subprocess.PIPE)
        kwargs.setdefault('stdout', subprocess.PIPE)
        kwargs.setdefault('stderr', sys.stderr)
        kwargs.setdefault('close_fds', True)
        kwargs.setdefault('shell', False)
        p = subprocess.Popen(execargs, **kwargs)
        stdoutdata, stderrdata = p.communicate(None)
        if p.returncode != 0:
            raise errors.CommandFailedError(
                "run_command %s exit %d:\n%s" %
                (execargs, p.returncode, stderrdata))
        return stdoutdata, stderrdata

    @staticmethod
    def git_checkout(url, version, path):
        if not re.search('^/', path):
            path = os.path.join(current_job().tmpdir, path)
        if not os.path.exists(path):
            util.run_command(["git", "clone", url, path],
                             cwd=os.path.dirname(path))
        util.run_command(["git", "checkout", version],
                         cwd=path)
        return path

    @staticmethod
    def tar_extractor(path, decompress_flag):
        return subprocess.Popen(["tar",
                                 "-C", path,
                                 ("-x%sf" % decompress_flag),
                                 "-"],
                                stdout=None,
                                stdin=subprocess.PIPE, stderr=sys.stderr,
                                shell=False, close_fds=True)

    @staticmethod
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
                if re.search('\.(tbz|tar.bz2)$', f.name()):
                    p = util.tar_extractor(path, 'j')
                elif re.search('\.(tgz|tar.gz)$', f.name()):
                    p = util.tar_extractor(path, 'z')
                elif re.search('\.tar$', f.name()):
                    p = util.tar_extractor(path, '')
                else:
                    raise errors.AssertionError(
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
                    raise errors.CommandFailedError(
                        "tar exited %d" % p.returncode)
            os.symlink(tarball, os.path.join(path, '.locator'))
        tld_extracts = filter(lambda f: f != '.locator', os.listdir(path))
        lockfile.close()
        if len(tld_extracts) == 1:
            return os.path.join(path, tld_extracts[0])
        return path

    @staticmethod
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
                if not re.search('\.zip$', f.name()):
                    raise errors.NotImplementedError(
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
                    raise errors.CommandFailedError(
                        "unzip exited %d" % p.returncode)
                os.unlink(zip_filename)
            os.symlink(zipball, os.path.join(path, '.locator'))
        tld_extracts = filter(lambda f: f != '.locator', os.listdir(path))
        lockfile.close()
        if len(tld_extracts) == 1:
            return os.path.join(path, tld_extracts[0])
        return path

    @staticmethod
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
                    util.mkdir_dash_p(os.path.dirname(os.path.join(path, stream_name, outname)))
                    outfile = open(os.path.join(path, stream_name, outname), 'wb')
                    for buf in (f.readall_decompressed() if decompress
                                else f.readall()):
                        outfile.write(buf)
                    outfile.close()
        if len(files_got) < len(files):
            raise errors.AssertionError(
                "Wanted files %s but only got %s from %s" %
                (files, files_got,
                 [z.name() for z in CollectionReader(collection).all_files()]))
        os.symlink(collection_hash, os.path.join(path, '.locator'))

        lockfile.close()
        return path

    @staticmethod
    def mkdir_dash_p(path):
        if not os.path.exists(path):
            util.mkdir_dash_p(os.path.dirname(path))
            try:
                os.mkdir(path)
            except OSError:
                if not os.path.exists(path):
                    os.mkdir(path)

    @staticmethod
    def stream_extract(stream, path, files=[], decompress=True):
        """Retrieve a stream from Keep and extract it to a local
        directory.  Return the absolute path where the stream was
        extracted.

        stream -- StreamReader object
        path -- where to extract: absolute, or relative to job tmp
        """
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
                util.mkdir_dash_p(os.path.dirname(os.path.join(path, outname)))
                outfile = open(os.path.join(path, outname), 'wb')
                for buf in (f.readall_decompressed() if decompress
                            else f.readall()):
                    outfile.write(buf)
                outfile.close()
        if len(files_got) < len(files):
            raise errors.AssertionError(
                "Wanted files %s but only got %s from %s" %
                (files, files_got, [z.name() for z in stream.all_files()]))
        lockfile.close()
        return path

    @staticmethod
    def listdir_recursive(dirname, base=None):
        allfiles = []
        for ent in sorted(os.listdir(dirname)):
            ent_path = os.path.join(dirname, ent)
            ent_base = os.path.join(base, ent) if base else ent
            if os.path.isdir(ent_path):
                allfiles += util.listdir_recursive(ent_path, ent_base)
            else:
                allfiles += [ent_base]
        return allfiles

