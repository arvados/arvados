# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados Python SDK

This module provides the entire Python SDK for Arvados. The most useful modules
include:

* arvados.api - After you `import arvados`, you can call `arvados.api` as a
  shortcut to the client constructor function `arvados.api.api`.

* arvados.collection - The `arvados.collection.Collection` class provides a
  high-level interface to read and write collections. It coordinates sending
  data to and from Keep, and synchronizing updates with the collection object.

* arvados.util - Utility functions to use mostly in conjunction with the API
  client object and the results it returns.

Other submodules provide lower-level functionality.
"""

import logging as stdliblog
import os
import sys
import types

from collections import UserDict

from . import api, errors, util
from .api import api_from_config, http_cache
from .collection import CollectionReader, CollectionWriter, ResumableCollectionWriter
from arvados.keep import *
from arvados.stream import *
from .arvfile import StreamFileReader
from .logging import log_format, log_date_format, log_handler
from .retry import RetryLoop

# Previous versions of the PySDK used to say `from .api import api`.  This
# made it convenient to call the API client constructor, but difficult to
# access the rest of the `arvados.api` module. The magic below fixes that
# bug while retaining backwards compatibility: `arvados.api` is now the
# module and you can import it normally, but we make that module callable so
# all the existing code that says `arvados.api('v1', ...)` still works.
class _CallableAPIModule(api.__class__):
    def __call__(self, *args, **kwargs):
        return self.api(*args, **kwargs)
api.__class__ = _CallableAPIModule

# Override logging module pulled in via `from ... import *`
# so users can `import arvados.logging`.
logging = sys.modules['arvados.logging']

# Set up Arvados logging based on the user's configuration.
# All Arvados code should log under the arvados hierarchy.
logger = stdliblog.getLogger('arvados')
logger.addHandler(log_handler)
logger.setLevel(stdliblog.DEBUG if config.get('ARVADOS_DEBUG')
                else stdliblog.WARNING)

@util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
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
@util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
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
@util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
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

@util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
def getjobparam(*args):
    return current_job()['script_parameters'].get(*args)

@util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
def get_job_param_mount(*args):
    return os.path.join(os.environ['TASK_KEEPMOUNT'], current_job()['script_parameters'].get(*args))

@util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
def get_task_param_mount(*args):
    return os.path.join(os.environ['TASK_KEEPMOUNT'], current_task()['parameters'].get(*args))

class JobTask(object):
    @util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
    def __init__(self, parameters=dict(), runtime_constraints=dict()):
        print("init jobtask %s %s" % (parameters, runtime_constraints))

class job_setup(object):
    @staticmethod
    @util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
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
    @util._deprecated('3.0', 'arvados-cwl-runner or the containers API')
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
