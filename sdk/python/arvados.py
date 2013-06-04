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
    _current_task = t
    return t

_current_job = None
def current_job():
    global _current_job
    if _current_job:
        return _current_job
    t = service.jobs().get(uuid=os.environ['JOB_UUID']).execute()
    _current_job = t
    return t

class JobTask:
    def __init__(self, parameters=dict(), resource_limits=dict()):
        print "init jobtask %s %s" % (parameters, resource_limits)

class job_setup:
    @staticmethod
    def one_task_per_input_file(if_sequence=0, and_end_task=True):
        if if_sequence != current_task()['sequence']:
            return
        job_input = current_job()['script_parameters']['input']
        p = subprocess.Popen(["whls", job_input],
                             stdout=subprocess.PIPE,
                             stdin=None, stderr=None,
                             shell=False, close_fds=True)
        for f in p.stdout.read().split("\n"):
            if f != '':
                task_input = job_input + '/' + re.sub(r'^\./', '', f)
                new_task_attrs = {
                    'job_uuid': current_job()['uuid'],
                    'created_by_job_task': current_task()['uuid'],
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
            raise Exception("subprocess exited %d" % self.p.returncode)
