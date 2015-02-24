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

from .api import api, http_cache
from collection import CollectionReader, CollectionWriter, ResumableCollectionWriter
from keep import *
from stream import *
from arvfile import StreamFileReader
import errors
import util

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

def get_job_param_mount(*args):
    return os.path.join(os.environ['TASK_KEEPMOUNT'], current_job()['script_parameters'].get(*args))

def get_task_param_mount(*args):
    return os.path.join(os.environ['TASK_KEEPMOUNT'], current_task()['parameters'].get(*args))

class JobTask(object):
    def __init__(self, parameters=dict(), runtime_constraints=dict()):
        print "init jobtask %s %s" % (parameters, runtime_constraints)

class job_setup:
    @staticmethod
    def one_task_per_input_file(if_sequence=0, and_end_task=True, input_as_path=False):
        if if_sequence != current_task()['sequence']:
            return
        job_input = current_job()['script_parameters']['input']
        cr = CollectionReader(job_input)
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
