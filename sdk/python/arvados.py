import gflags
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

from apiclient import errors
from apiclient.discovery import build

class CredentialsFromEnv:
    @staticmethod
    def http_request(self, uri, **kwargs):
        from httplib import BadStatusLine
        if 'headers' not in kwargs:
            kwargs['headers'] = {}
        kwargs['headers']['Authorization'] = 'OAuth2 %s' % os.environ['ARVADOS_API_TOKEN']
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

url = ('https://%s/discovery/v1/apis/'
       '{api}/{apiVersion}/rest' % os.environ['ARVADOS_API_HOST'])
credentials = CredentialsFromEnv()
http = httplib2.Http()
http = credentials.authorize(http)
http.disable_ssl_certificate_validation=True
service = build("arvados", "v1", http=http, discoveryServiceUrl=url)

def task_set_output(self,s):
    service.job_tasks().update(uuid=self['uuid'],
                               job_task=json.dumps({
                'output':s,
                'success':True,
                'progress':1.0
                })).execute()

_current_task = None
def current_task():
    global _current_task
    if _current_task:
        return _current_task
    t = service.job_tasks().get(uuid=os.environ['TASK_UUID']).execute()
    t = UserDict.UserDict(t)
    t.set_output = types.MethodType(task_set_output, t)
    t.tmpdir = os.environ['TASK_TMPDIR']
    _current_task = t
    return t

_current_job = None
def current_job():
    global _current_job
    if _current_job:
        return _current_job
    t = service.jobs().get(uuid=os.environ['JOB_UUID']).execute()
    t = UserDict.UserDict(t)
    t.tmpdir = os.environ['CRUNCH_WORK']
    _current_job = t
    return t

def api():
    return service

class JobTask:
    def __init__(self, parameters=dict(), resource_limits=dict()):
        print "init jobtask %s %s" % (parameters, resource_limits)

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
                service.job_tasks().create(job_task=json.dumps(new_task_attrs)).execute()
        if and_end_task:
            service.job_tasks().update(uuid=current_task()['uuid'],
                                       job_task=json.dumps({'success':True})
                                       ).execute()
            exit(0)

class util:
    @staticmethod
    def run_command(execargs, **kwargs):
        p = subprocess.Popen(execargs, close_fds=True, shell=False,
                             stdin=subprocess.PIPE,
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             **kwargs)
        stdoutdata, stderrdata = p.communicate(None)
        if p.returncode != 0:
            raise Exception("run_command %s exit %d:\n%s" %
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

class DataReader:
    def __init__(self, data_locator):
        self.data_locator = data_locator
        self.p = subprocess.Popen(["whget", "-r", self.data_locator, "-"],
                                  stdout=subprocess.PIPE,
                                  stdin=None, stderr=subprocess.PIPE,
                                  shell=False, close_fds=True)
    def __enter__(self):
        pass
    def __exit__(self):
        self.close()
    def read(self, size, **kwargs):
        return self.p.stdout.read(size, **kwargs)
    def close(self):
        self.p.stdout.close()
        if not self.p.stderr.closed:
            for err in self.p.stderr:
                print >> sys.stderr, err
            self.p.stderr.close()
        self.p.wait()
        if self.p.returncode != 0:
            raise Exception("whget subprocess exited %d" % self.p.returncode)

class StreamFileReader:
    def __init__(self, stream, pos, size, name):
        self._stream = stream
        self._pos = pos
        self._size = size
        self._name = name
        self._filepos = 0
    def name(self):
        return self._name
    def decompressed_name(self):
        return re.sub('\.(bz2|gz)$', '', self._name)
    def size(self):
        return self._size
    def stream_name(self):
        return self._stream.name()
    def read(self, size, **kwargs):
        self._stream.seek(self._pos + self._filepos)
        data = self._stream.read(min(size, self._size - self._filepos))
        self._filepos += len(data)
        return data
    def readall(self, size, **kwargs):
        while True:
            data = self.read(size, **kwargs)
            if data == '':
                break
            yield data
    def bunzip2(self, size):
        decompressor = bz2.BZ2Decompressor()
        for chunk in self.readall(size):
            data = decompressor.decompress(chunk)
            if data and data != '':
                yield data
    def gunzip(self, size):
        decompressor = zlib.decompressobj(16+zlib.MAX_WBITS)
        for chunk in self.readall(size):
            data = decompressor.decompress(decompressor.unconsumed_tail + chunk)
            if data and data != '':
                yield data
    def readlines(self, decompress=True):
        self._stream.seek(self._pos + self._filepos)
        if decompress and re.search('\.bz2$', self._name):
            datasource = self.bunzip2(2**10)
        elif decompress and re.search('\.gz$', self._name):
            datasource = self.gunzip(2**10)
        else:
            datasource = self.readall(2**10)
        data = ''
        for newdata in datasource:
            data += newdata
            sol = 0
            while True:
                eol = string.find(data, "\n", sol)
                if eol < 0:
                    break
                yield data[sol:eol+1]
                sol = eol+1
            data = data[sol:]
        if data != '':
            yield data
    def as_manifest(self):
        if self.size() == 0:
            return ("%s d41d8cd98f00b204e9800998ecf8427e+0 0:0:%s\n"
                    % (self._stream.name(), self.name()))
        return string.join(self._stream.tokens_for_range(self._pos, self._size),
                           " ") + "\n"

class StreamReader:
    def __init__(self, tokens):
        self._tokens = tokens
        self._current_datablock_data = None
        self._current_datablock_pos = 0
        self._current_datablock_index = -1
        self._pos = 0

        self._stream_name = None
        self.data_locators = []
        self.files = []

        for tok in self._tokens:
            if self._stream_name == None:
                self._stream_name = tok
            elif re.search(r'^[0-9a-f]{32}(\+\S+)*$', tok):
                self.data_locators += [tok]
            elif re.search(r'^\d+:\d+:\S+', tok):
                pos, size, name = tok.split(':',2)
                self.files += [[int(pos), int(size), name]]
            else:
                raise Exception("Invalid manifest format")
    def tokens_for_range(self, range_start, range_size):
        resp = [self._stream_name]
        return_all_tokens = False
        block_start = 0
        token_bytes_skipped = 0
        for locator in self.data_locators:
            sizehint = re.search(r'\+(\d+)', locator)
            if not sizehint:
                return_all_tokens = True
            if return_all_tokens:
                resp += [locator]
                next
            blocksize = int(sizehint.group(0))
            if range_start + range_size <= block_start:
                break
            if range_start < block_start + blocksize:
                resp += [locator]
            else:
                token_bytes_skipped += blocksize
            block_start += blocksize
        for f in self.files:
            if ((f[0] < range_start + range_size)
                and
                (f[0] + f[1] > range_start)
                and
                f[1] > 0):
                resp += ["%d:%d:%s" % (f[0] - token_bytes_skipped, f[1], f[2])]
        return resp
    def name(self):
        return self._stream_name
    def all_files(self):
        for f in self.files:
            pos, size, name = f
            yield StreamFileReader(self, pos, size, name)
    def nextdatablock(self):
        if self._current_datablock_index < 0:
            self._current_datablock_pos = 0
            self._current_datablock_index = 0
        else:
            self._current_datablock_pos += self.current_datablock_size()
            self._current_datablock_index += 1
        self._current_datablock_data = None
    def current_datablock_data(self):
        if self._current_datablock_data == None:
            self._current_datablock_data = Keep.get(self.data_locators[self._current_datablock_index])
        return self._current_datablock_data
    def current_datablock_size(self):
        if self._current_datablock_index < 0:
            self.nextdatablock()
        sizehint = re.search('\+(\d+)', self.data_locators[self._current_datablock_index])
        if sizehint:
            return int(sizehint.group(0))
        return len(self.current_datablock_data())
    def seek(self, pos):
        """Set the position of the next read operation."""
        self._pos = pos
    def really_seek(self):
        """Find and load the appropriate data block, so the byte at
        _pos is in memory.
        """
        if self._pos == self._current_datablock_pos:
            return True
        if (self._current_datablock_pos != None and
            self._pos >= self._current_datablock_pos and
            self._pos <= self._current_datablock_pos + self.current_datablock_size()):
            return True
        if self._pos < self._current_datablock_pos:
            self._current_datablock_index = -1
            self.nextdatablock()
        while (self._pos > self._current_datablock_pos and
               self._pos > self._current_datablock_pos + self.current_datablock_size()):
            self.nextdatablock()
    def read(self, size):
        """Read no more than size bytes -- but at least one byte,
        unless _pos is already at the end of the stream.
        """
        if size == 0:
            return ''
        self.really_seek()
        while self._pos >= self._current_datablock_pos + self.current_datablock_size():
            self.nextdatablock()
            if self._current_datablock_index >= len(self.data_locators):
                return None
        data = self.current_datablock_data()[self._pos - self._current_datablock_pos : self._pos - self._current_datablock_pos + size]
        self._pos += len(data)
        return data

class CollectionReader:
    def __init__(self, manifest_locator_or_text):
        if re.search(r'^\S+( [a-f0-9]{32,}(\+\S+)*)+( \d+:\d+:\S+)+\n', manifest_locator_or_text):
            self._manifest_text = manifest_locator_or_text
            self._manifest_locator = None
        else:
            self._manifest_locator = manifest_locator_or_text
            self._manifest_text = None
        self._streams = None
    def __enter__(self):
        pass
    def __exit__(self):
        pass
    def _populate(self):
        if self._streams != None:
            return
        if not self._manifest_text:
            self._manifest_text = Keep.get(self._manifest_locator)
        self._streams = []
        for stream_line in self._manifest_text.split("\n"):
            stream_tokens = stream_line.split()
            self._streams += [stream_tokens]
    def all_streams(self):
        self._populate()
        resp = []
        for s in self._streams:
            resp += [StreamReader(s)]
        return resp
    def all_files(self):
        for s in self.all_streams():
            for f in s.all_files():
                yield f

class CollectionWriter:
    KEEP_BLOCK_SIZE = 2**26
    def __init__(self):
        self._data_buffer = []
        self._data_buffer_len = 0
        self._current_stream_files = []
        self._current_stream_length = 0
        self._current_stream_locators = []
        self._current_stream_name = '.'
        self._current_file_name = None
        self._current_file_pos = 0
        self._finished_streams = []
    def __enter__(self):
        pass
    def __exit__(self):
        self.finish()
    def write(self, newdata):
        self._data_buffer += [newdata]
        self._data_buffer_len += len(newdata)
        self._current_stream_length += len(newdata)
        while self._data_buffer_len >= self.KEEP_BLOCK_SIZE:
            self.flush_data()
    def flush_data(self):
        data_buffer = ''.join(self._data_buffer)
        if data_buffer != '':
            self._current_stream_locators += [Keep.put(data_buffer[0:self.KEEP_BLOCK_SIZE])]
            self._data_buffer = [data_buffer[self.KEEP_BLOCK_SIZE:]]
    def start_new_file(self, newfilename=None):
        self.finish_current_file()
        self.set_current_file_name(newfilename)
    def set_current_file_name(self, newfilename):
        if re.search(r'[ \t\n]', newfilename):
            raise AssertionError("Manifest filenames cannot contain whitespace")
        self._current_file_name = newfilename
    def current_file_name(self):
        return self._current_file_name
    def finish_current_file(self):
        if self._current_file_name == None:
            if self._current_file_pos == self._current_stream_length:
                return
            raise Exception("Cannot finish an unnamed file (%d bytes at offset %d in '%s' stream)" % (self._current_stream_length - self._current_file_pos, self._current_file_pos, self._current_stream_name))
        self._current_stream_files += [[self._current_file_pos,
                                       self._current_stream_length - self._current_file_pos,
                                       self._current_file_name]]
        self._current_file_pos = self._current_stream_length
    def start_new_stream(self, newstreamname=None):
        self.finish_current_stream()
        self.set_current_stream_name(newstreamname)
    def set_current_stream_name(self, newstreamname):
        if re.search(r'[ \t\n]', newstreamname):
            raise AssertionError("Manifest stream names cannot contain whitespace")
        self._current_stream_name = newstreamname
    def current_stream_name(self):
        return self._current_stream_name
    def finish_current_stream(self):
        self.finish_current_file()
        self.flush_data()
        if len(self._current_stream_files) == 0:
            pass
        elif self._current_stream_name == None:
            raise Exception("Cannot finish an unnamed stream (%d bytes in %d files)" % (self._current_stream_length, len(self._current_stream_files)))
        else:
            self._finished_streams += [[self._current_stream_name,
                                       self._current_stream_locators,
                                       self._current_stream_files]]
        self._current_stream_files = []
        self._current_stream_length = 0
        self._current_stream_locators = []
        self._current_stream_name = None
        self._current_file_pos = 0
        self._current_file_name = None
    def finish(self):
        return Keep.put(self.manifest_text())
    def manifest_text(self):
        self.finish_current_stream()
        manifest = ''
        for stream in self._finished_streams:
            manifest += stream[0]
            if len(stream[1]) == 0:
                manifest += " d41d8cd98f00b204e9800998ecf8427e+0"
            else:
                for locator in stream[1]:
                    manifest += " %s" % locator
            for sfile in stream[2]:
                manifest += " %d:%d:%s" % (sfile[0], sfile[1], sfile[2])
            manifest += "\n"
        return manifest

class Keep:
    @staticmethod
    def put(data):
        if 'KEEP_LOCAL_STORE' in os.environ:
            return Keep.local_store_put(data)
        p = subprocess.Popen(["whput", "-"],
                             stdout=subprocess.PIPE,
                             stdin=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             shell=False, close_fds=True)
        stdoutdata, stderrdata = p.communicate(data)
        if p.returncode != 0:
            raise Exception("whput subprocess exited %d - stderr:\n%s" % (p.returncode, stderrdata))
        return stdoutdata.rstrip()
    @staticmethod
    def get(locator):
        if 'KEEP_LOCAL_STORE' in os.environ:
            return Keep.local_store_get(locator)
        p = subprocess.Popen(["whget", locator, "-"],
                             stdout=subprocess.PIPE,
                             stdin=None,
                             stderr=subprocess.PIPE,
                             shell=False, close_fds=True)
        stdoutdata, stderrdata = p.communicate(None)
        if p.returncode != 0:
            raise Exception("whget subprocess exited %d - stderr:\n%s" % (p.returncode, stderrdata))
        m = hashlib.new('md5')
        m.update(stdoutdata)
        try:
            if locator.index(m.hexdigest()) == 0:
                return stdoutdata
        except ValueError:
            pass
        raise Exception("md5 checksum mismatch: md5(get(%s)) == %s" % (locator, m.hexdigest()))
    @staticmethod
    def local_store_put(data):
        m = hashlib.new('md5')
        m.update(data)
        md5 = m.hexdigest()
        locator = '%s+%d' % (md5, len(data))
        with open(os.path.join(os.environ['KEEP_LOCAL_STORE'], md5 + '.tmp'), 'w') as f:
            f.write(data)
        os.rename(os.path.join(os.environ['KEEP_LOCAL_STORE'], md5 + '.tmp'),
                  os.path.join(os.environ['KEEP_LOCAL_STORE'], md5))
        return locator
    @staticmethod
    def local_store_get(locator):
        r = re.search('^([0-9a-f]{32,})', locator)
        if not r:
            raise Exception("Keep.get: invalid data locator '%s'" % locator)
        if r.group(0) == 'd41d8cd98f00b204e9800998ecf8427e':
            return ''
        with open(os.path.join(os.environ['KEEP_LOCAL_STORE'], r.group(0)), 'r') as f:
            return f.read()
