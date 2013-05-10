import gflags
import httplib2
import logging
import os
import pprint
import sys
import types

from apiclient import errors
from apiclient.discovery import build

class CredentialsFromEnv:
    @staticmethod
    def http_request(self, uri, **kwargs):
        if 'headers' not in kwargs:
            kwargs['headers'] = {}
        kwargs['headers']['Authorization'] = 'OAuth2 %s' % os.environ['ARVADOS_API_TOKEN']
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

def current_task():
    t = service.job_tasks().get(uuid=os.environ['TASK_UUID']).execute()
    return t

def current_job():
    t = service.jobs().get(uuid=os.environ['JOB_UUID']).execute()
    return t

class JobTask:
    def __init__(self, parameters=dict(), resource_limits=dict()):
        print "init jobtask %s %s" % (parameters, resource_limits)

class job_setup:
    @staticmethod
    def one_task_per_input_file(if_sequence=0, and_end_task=True):
        if if_sequence != current_job()['sequence']:
            return
        job_input = current_job()['parameters']['input']
        p = subprocess.Popen(["whls", job_input],
                             stdout=subprocess.PIPE,
                             stdin=None, stderr=None,
                             shell=False, close_fds=True)
        for f in p.stdout.read().split("\n"):
            if f != '':
                task_input = job_input + '/' + f
                new_task_attrs = {
                    'job_uuid': current_job()['uuid'],
                    'parameters': {
                        'input':task_input
                        }
                    }
                service.jobs_tasks().create(job_task=new_task_attrs)
        if and_end_task:
            service.job_tasks().update(uuid=current_task()['uuid'],
                                       job_task={'success':True})
            exit 0
