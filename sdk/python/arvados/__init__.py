from __future__ import print_function
from __future__ import absolute_import
from future import standard_library
standard_library.install_aliases()
from builtins import object
import bz2
import fcntl
import hashlib
import http.client
import httplib2
import json
import logging
import os
import pprint
import re
import string
import subprocess
import sys
import threading
import time
import types
import zlib

if sys.version_info >= (3, 0):
    from collections import UserDict
else:
    from UserDict import UserDict

from .api import api, api_from_config, http_cache
from .collection import CollectionReader, CollectionWriter, ResumableCollectionWriter
from arvados.keep import *
from arvados.stream import *
from .arvfile import StreamFileReader
from .retry import RetryLoop
import arvados.errors as errors
import arvados.util as util

# Set up Arvados logging based on the user's configuration.
# All Arvados code should log under the arvados hierarchy.
log_handler = logging.StreamHandler()
log_handler.setFormatter(logging.Formatter(
        '%(asctime)s %(name)s[%(process)d] %(levelname)s: %(message)s',
        '%Y-%m-%d %H:%M:%S'))
logger = logging.getLogger('arvados')
logger.addHandler(log_handler)
logger.setLevel(logging.DEBUG if config.get('ARVADOS_DEBUG')
                else logging.WARNING)

def task_set_output(self, s, num_retries=5):
    for tries_left in RetryLoop(num_retries=num_retries, backoff_start=0):
        try:
            return api('v1').job_tasks().update(
                uuid=self['uuid'],
                body={
                    'output':s,
                    'success':True,
                    'progress':1.0
                }).execute()
        except errors.ApiError as error:
            if retry.check_http_response_success(error.resp.status) is None and tries_left > 0:
                logger.debug("task_set_output: job_tasks().update() raised {}, retrying with {} tries left".format(repr(error),tries_left))
            else:
                raise

_current_task = None
def current_task(num_retries=5):
    global _current_task
    if _current_task:
        return _current_task

    for tries_left in RetryLoop(num_retries=num_retries, backoff_start=2):
        try:
            task = api('v1').job_tasks().get(uuid=os.environ['TASK_UUID']).execute()
            task = UserDict(task)
            task.set_output = types.MethodType(task_set_output, task)
            task.tmpdir = os.environ['TASK_WORK']
            _current_task = task
            return task
        except errors.ApiError as error:
            if retry.check_http_response_success(error.resp.status) is None and tries_left > 0:
                logger.debug("current_task: job_tasks().get() raised {}, retrying with {} tries left".format(repr(error),tries_left))
            else:
                raise

_current_job = None
def current_job(num_retries=5):
    global _current_job
    if _current_job:
        return _current_job

    for tries_left in RetryLoop(num_retries=num_retries, backoff_start=2):
        try:
            job = api('v1').jobs().get(uuid=os.environ['JOB_UUID']).execute()
            job = UserDict(job)
            job.tmpdir = os.environ['JOB_WORK']
            _current_job = job
            return job
        except errors.ApiError as error:
            if retry.check_http_response_success(error.resp.status) is None and tries_left > 0:
                logger.debug("current_job: jobs().get() raised {}, retrying with {} tries left".format(repr(error),tries_left))
            else:
                raise

def getjobparam(*args):
    return current_job()['script_parameters'].get(*args)

def get_job_param_mount(*args):
    return os.path.join(os.environ['TASK_KEEPMOUNT'], current_job()['script_parameters'].get(*args))

def get_task_param_mount(*args):
    return os.path.join(os.environ['TASK_KEEPMOUNT'], current_task()['parameters'].get(*args))

class JobTask(object):
    def __init__(self, parameters=dict(), runtime_constraints=dict()):
        print("init jobtask %s %s" % (parameters, runtime_constraints))

class job_setup(object):
    @staticmethod
    def one_task_per_input_file(if_sequence=0, and_end_task=True, input_as_path=False, api_client=None):
        if if_sequence != current_task()['sequence']:
            return

        if not api_client:
            api_client = api('v1')

        job_input = current_job()['script_parameters']['input']
        cr = CollectionReader(job_input, api_client=api_client)
        cr.normalize()
        for s in cr.all_streams():
            for f in s.all_files():
                if input_as_path:
                    task_input = os.path.join(job_input, s.name(), f.name())
                else:
                    task_input = f.as_manifest()
                new_task_attrs = {
                    'job_uuid': current_job()['uuid'],
                    'created_by_job_task_uuid': current_task()['uuid'],
                    'sequence': if_sequence + 1,
                    'parameters': {
                        'input':task_input
                        }
                    }
                api_client.job_tasks().create(body=new_task_attrs).execute()
        if and_end_task:
            api_client.job_tasks().update(uuid=current_task()['uuid'],
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
