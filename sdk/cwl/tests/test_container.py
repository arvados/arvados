# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from builtins import str
from builtins import object

import arvados_cwl
import arvados_cwl.context
import arvados_cwl.util
from arvados_cwl.arvdocker import arv_docker_clear_cache
import copy
import arvados.config
import logging
import mock
import unittest
import os
import functools
import cwltool.process
import cwltool.secrets
from cwltool.update import INTERNAL_VERSION
from schema_salad.ref_resolver import Loader
from schema_salad.sourceline import cmap

from .matcher import JsonDiffMatcher, StripYAMLComments
from .mock_discovery import get_rootDesc

if not os.getenv('ARVADOS_DEBUG'):
    logging.getLogger('arvados.cwl-runner').setLevel(logging.WARN)
    logging.getLogger('arvados.arv-run').setLevel(logging.WARN)

class CollectionMock(object):
    def __init__(self, vwdmock, *args, **kwargs):
        self.vwdmock = vwdmock
        self.count = 0

    def open(self, *args, **kwargs):
        self.count += 1
        return self.vwdmock.open(*args, **kwargs)

    def copy(self, *args, **kwargs):
        self.count += 1
        self.vwdmock.copy(*args, **kwargs)

    def save_new(self, *args, **kwargs):
        pass

    def __len__(self):
        return self.count

    def portable_data_hash(self):
        if self.count == 0:
            return arvados.config.EMPTY_BLOCK_LOCATOR
        else:
            return "99999999999999999999999999999996+99"


class TestContainer(unittest.TestCase):

    def setUp(self):
        cwltool.process._names = set()
        arv_docker_clear_cache()

    def helper(self, runner, enable_reuse=True):
        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema(INTERNAL_VERSION)

        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess,
                                         collection_cache=arvados_cwl.CollectionCache(runner.api, None, 0))
        loadingContext = arvados_cwl.context.ArvLoadingContext(
            {"avsc_names": avsc_names,
             "basedir": "",
             "make_fs_access": make_fs_access,
             "loader": Loader({}),
             "metadata": {"cwlVersion": INTERNAL_VERSION, "http://commonwl.org/cwltool#original_cwlVersion": "v1.0"}})
        runtimeContext = arvados_cwl.context.ArvRuntimeContext(
            {"work_api": "containers",
             "basedir": "",
             "name": "test_run_"+str(enable_reuse),
             "make_fs_access": make_fs_access,
             "tmpdir": "/tmp",
             "enable_reuse": enable_reuse,
             "priority": 500,
             "project_uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
            })

        return loadingContext, runtimeContext

    # Helper function to set up the ArvCwlExecutor to use the containers api
    # and test that the RuntimeStatusLoggingHandler is set up correctly
    def setup_and_test_container_executor_and_logging(self, gcc_mock) :
        api = mock.MagicMock()
        api._rootDesc = copy.deepcopy(get_rootDesc())

        # Make sure ArvCwlExecutor thinks it's running inside a container so it
        # adds the logging handler that will call runtime_status_update() mock
        self.assertFalse(gcc_mock.called)
        runner = arvados_cwl.ArvCwlExecutor(api)
        self.assertEqual(runner.work_api, 'containers')
        root_logger = logging.getLogger('')
        handlerClasses = [h.__class__ for h in root_logger.handlers]
        self.assertTrue(arvados_cwl.RuntimeStatusLoggingHandler in handlerClasses)
        return runner

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_run(self, keepdocker):
        for enable_reuse in (True, False):
            arv_docker_clear_cache()

            runner = mock.MagicMock()
            runner.ignore_docker_for_reuse = False
            runner.intermediate_output_ttl = 0
            runner.secret_store = cwltool.secrets.SecretStore()

            keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
            runner.api.collections().get().execute.return_value = {
                "portable_data_hash": "99999999999999999999999999999993+99"}

            tool = cmap({
                "inputs": [],
                "outputs": [],
                "baseCommand": "ls",
                "arguments": [{"valueFrom": "$(runtime.outdir)"}],
                "id": "#",
                "class": "CommandLineTool"
            })

            loadingContext, runtimeContext = self.helper(runner, enable_reuse)

            arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, loadingContext)
            arvtool.formatgraph = None

            for j in arvtool.job({}, mock.MagicMock(), runtimeContext):
                j.run(runtimeContext)
                runner.api.container_requests().create.assert_called_with(
                    body=JsonDiffMatcher({
                        'environment': {
                            'HOME': '/var/spool/cwl',
                            'TMPDIR': '/tmp'
                        },
                        'name': 'test_run_'+str(enable_reuse),
                        'runtime_constraints': {
                            'vcpus': 1,
                            'ram': 1073741824
                        },
                        'use_existing': enable_reuse,
                        'priority': 500,
                        'mounts': {
                            '/tmp': {'kind': 'tmp',
                                     "capacity": 1073741824
                                 },
                            '/var/spool/cwl': {'kind': 'tmp',
                                               "capacity": 1073741824 }
                        },
                        'state': 'Committed',
                        'output_name': 'Output for step test_run_'+str(enable_reuse),
                        'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                        'output_path': '/var/spool/cwl',
                        'output_ttl': 0,
                        'container_image': '99999999999999999999999999999993+99',
                        'command': ['ls', '/var/spool/cwl'],
                        'cwd': '/var/spool/cwl',
                        'scheduling_parameters': {},
                        'properties': {},
                        'secret_mounts': {}
                    }))

    # The test passes some fields in builder.resources
    # For the remaining fields, the defaults will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_resource_requirements(self, keepdocker):
        runner = mock.MagicMock()
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 3600
        runner.secret_store = cwltool.secrets.SecretStore()

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        tool = cmap({
            "inputs": [],
            "outputs": [],
            "hints": [{
                "class": "ResourceRequirement",
                "coresMin": 3,
                "ramMin": 3000,
                "tmpdirMin": 4000,
                "outdirMin": 5000
            }, {
                "class": "http://arvados.org/cwl#RuntimeConstraints",
                "keep_cache": 512
            }, {
                "class": "http://arvados.org/cwl#APIRequirement",
            }, {
                "class": "http://arvados.org/cwl#PartitionRequirement",
                "partition": "blurb"
            }, {
                "class": "http://arvados.org/cwl#IntermediateOutput",
                "outputTTL": 7200
            }, {
                "class": "http://arvados.org/cwl#ReuseRequirement",
                "enableReuse": False
            }],
            "baseCommand": "ls",
            "id": "#",
            "class": "CommandLineTool"
        })

        loadingContext, runtimeContext = self.helper(runner)
        runtimeContext.name = "test_resource_requirements"

        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, loadingContext)
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), runtimeContext):
            j.run(runtimeContext)

        call_args, call_kwargs = runner.api.container_requests().create.call_args

        call_body_expected = {
            'environment': {
                'HOME': '/var/spool/cwl',
                'TMPDIR': '/tmp'
            },
            'name': 'test_resource_requirements',
            'runtime_constraints': {
                'vcpus': 3,
                'ram': 3145728000,
                'keep_cache_ram': 536870912,
                'API': True
            },
            'use_existing': False,
            'priority': 500,
            'mounts': {
                '/tmp': {'kind': 'tmp',
                         "capacity": 4194304000 },
                '/var/spool/cwl': {'kind': 'tmp',
                                   "capacity": 5242880000 }
            },
            'state': 'Committed',
            'output_name': 'Output for step test_resource_requirements',
            'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
            'output_path': '/var/spool/cwl',
            'output_ttl': 7200,
            'container_image': '99999999999999999999999999999993+99',
            'command': ['ls'],
            'cwd': '/var/spool/cwl',
            'scheduling_parameters': {
                'partitions': ['blurb']
            },
            'properties': {},
            'secret_mounts': {}
        }

        call_body = call_kwargs.get('body', None)
        self.assertNotEqual(None, call_body)
        for key in call_body:
            self.assertEqual(call_body_expected.get(key), call_body.get(key))


    # The test passes some fields in builder.resources
    # For the remaining fields, the defaults will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    @mock.patch("arvados.collection.Collection")
    def test_initial_work_dir(self, collection_mock, keepdocker):
        runner = mock.MagicMock()
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0
        runner.secret_store = cwltool.secrets.SecretStore()

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        sourcemock = mock.MagicMock()
        def get_collection_mock(p):
            if "/" in p:
                return (sourcemock, p.split("/", 1)[1])
            else:
                return (sourcemock, "")
        runner.fs_access.get_collection.side_effect = get_collection_mock

        vwdmock = mock.MagicMock()
        collection_mock.side_effect = lambda *args, **kwargs: CollectionMock(vwdmock, *args, **kwargs)

        tool = cmap({
            "inputs": [],
            "outputs": [],
            "hints": [{
                "class": "InitialWorkDirRequirement",
                "listing": [{
                    "class": "File",
                    "basename": "foo",
                    "location": "keep:99999999999999999999999999999995+99/bar"
                },
                {
                    "class": "Directory",
                    "basename": "foo2",
                    "location": "keep:99999999999999999999999999999995+99"
                },
                {
                    "class": "File",
                    "basename": "filename",
                    "location": "keep:99999999999999999999999999999995+99/baz/filename"
                },
                {
                    "class": "Directory",
                    "basename": "subdir",
                    "location": "keep:99999999999999999999999999999995+99/subdir"
                }                        ]
            }],
            "baseCommand": "ls",
            "id": "#",
            "class": "CommandLineTool"
        })

        loadingContext, runtimeContext = self.helper(runner)
        runtimeContext.name = "test_initial_work_dir"

        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, loadingContext)
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), runtimeContext):
            j.run(runtimeContext)

        call_args, call_kwargs = runner.api.container_requests().create.call_args

        vwdmock.copy.assert_has_calls([mock.call('bar', 'foo', source_collection=sourcemock)])
        vwdmock.copy.assert_has_calls([mock.call('.', 'foo2', source_collection=sourcemock)])
        vwdmock.copy.assert_has_calls([mock.call('baz/filename', 'filename', source_collection=sourcemock)])
        vwdmock.copy.assert_has_calls([mock.call('subdir', 'subdir', source_collection=sourcemock)])

        call_body_expected = {
            'environment': {
                'HOME': '/var/spool/cwl',
                'TMPDIR': '/tmp'
            },
            'name': 'test_initial_work_dir',
            'runtime_constraints': {
                'vcpus': 1,
                'ram': 1073741824
            },
            'use_existing': True,
            'priority': 500,
            'mounts': {
                '/tmp': {'kind': 'tmp',
                         "capacity": 1073741824 },
                '/var/spool/cwl': {'kind': 'tmp',
                                   "capacity": 1073741824 },
                '/var/spool/cwl/foo': {
                    'kind': 'collection',
                    'path': 'foo',
                    'portable_data_hash': '99999999999999999999999999999996+99'
                },
                '/var/spool/cwl/foo2': {
                    'kind': 'collection',
                    'path': 'foo2',
                    'portable_data_hash': '99999999999999999999999999999996+99'
                },
                '/var/spool/cwl/filename': {
                    'kind': 'collection',
                    'path': 'filename',
                    'portable_data_hash': '99999999999999999999999999999996+99'
                },
                '/var/spool/cwl/subdir': {
                    'kind': 'collection',
                    'path': 'subdir',
                    'portable_data_hash': '99999999999999999999999999999996+99'
                }
            },
            'state': 'Committed',
            'output_name': 'Output for step test_initial_work_dir',
            'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
            'output_path': '/var/spool/cwl',
            'output_ttl': 0,
            'container_image': '99999999999999999999999999999993+99',
            'command': ['ls'],
            'cwd': '/var/spool/cwl',
            'scheduling_parameters': {
            },
            'properties': {},
            'secret_mounts': {}
        }

        call_body = call_kwargs.get('body', None)
        self.assertNotEqual(None, call_body)
        for key in call_body:
            self.assertEqual(call_body_expected.get(key), call_body.get(key))


    # Test redirecting stdin/stdout/stderr
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_redirects(self, keepdocker):
        runner = mock.MagicMock()
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0
        runner.secret_store = cwltool.secrets.SecretStore()

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema(INTERNAL_VERSION)

        tool = cmap({
            "inputs": [],
            "outputs": [],
            "baseCommand": "ls",
            "stdout": "stdout.txt",
            "stderr": "stderr.txt",
            "stdin": "/keep/99999999999999999999999999999996+99/file.txt",
            "arguments": [{"valueFrom": "$(runtime.outdir)"}],
            "id": "#",
            "class": "CommandLineTool"
        })

        loadingContext, runtimeContext = self.helper(runner)
        runtimeContext.name = "test_run_redirect"

        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, loadingContext)
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), runtimeContext):
            j.run(runtimeContext)
            runner.api.container_requests().create.assert_called_with(
                body=JsonDiffMatcher({
                    'environment': {
                        'HOME': '/var/spool/cwl',
                        'TMPDIR': '/tmp'
                    },
                    'name': 'test_run_redirect',
                    'runtime_constraints': {
                        'vcpus': 1,
                        'ram': 1073741824
                    },
                    'use_existing': True,
                    'priority': 500,
                    'mounts': {
                        '/tmp': {'kind': 'tmp',
                                 "capacity": 1073741824 },
                        '/var/spool/cwl': {'kind': 'tmp',
                                           "capacity": 1073741824 },
                        "stderr": {
                            "kind": "file",
                            "path": "/var/spool/cwl/stderr.txt"
                        },
                        "stdin": {
                            "kind": "collection",
                            "path": "file.txt",
                            "portable_data_hash": "99999999999999999999999999999996+99"
                        },
                        "stdout": {
                            "kind": "file",
                            "path": "/var/spool/cwl/stdout.txt"
                        },
                    },
                    'state': 'Committed',
                    "output_name": "Output for step test_run_redirect",
                    'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                    'output_path': '/var/spool/cwl',
                    'output_ttl': 0,
                    'container_image': '99999999999999999999999999999993+99',
                    'command': ['ls', '/var/spool/cwl'],
                    'cwd': '/var/spool/cwl',
                    'scheduling_parameters': {},
                    'properties': {},
                    'secret_mounts': {}
                }))

    @mock.patch("arvados.collection.Collection")
    def test_done(self, col):
        api = mock.MagicMock()

        runner = mock.MagicMock()
        runner.api = api
        runner.num_retries = 0
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0
        runner.secret_store = cwltool.secrets.SecretStore()

        runner.api.containers().get().execute.return_value = {"state":"Complete",
                                                              "output": "abc+123",
                                                              "exit_code": 0}

        col().open.return_value = []

        loadingContext, runtimeContext = self.helper(runner)

        arvjob = arvados_cwl.ArvadosContainer(runner,
                                              runtimeContext,
                                              mock.MagicMock(),
                                              {},
                                              None,
                                              [],
                                              [],
                                              "testjob")
        arvjob.output_callback = mock.MagicMock()
        arvjob.collect_outputs = mock.MagicMock()
        arvjob.successCodes = [0]
        arvjob.outdir = "/var/spool/cwl"
        arvjob.output_ttl = 3600

        arvjob.collect_outputs.return_value = {"out": "stuff"}

        arvjob.done({
            "state": "Final",
            "log_uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz1",
            "output_uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2",
            "uuid": "zzzzz-xvhdp-zzzzzzzzzzzzzzz",
            "container_uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz",
            "modified_at": "2017-05-26T12:01:22Z"
        })

        self.assertFalse(api.collections().create.called)
        self.assertFalse(runner.runtime_status_error.called)

        arvjob.collect_outputs.assert_called_with("keep:abc+123", 0)
        arvjob.output_callback.assert_called_with({"out": "stuff"}, "success")
        runner.add_intermediate_output.assert_called_with("zzzzz-4zz18-zzzzzzzzzzzzzz2")

    # Test to make sure we dont call runtime_status_update if we already did
    # some where higher up in the call stack
    @mock.patch("arvados_cwl.util.get_current_container")
    def test_recursive_runtime_status_update(self, gcc_mock):
        self.setup_and_test_container_executor_and_logging(gcc_mock)
        root_logger = logging.getLogger('')

        # get_current_container is invoked when we call runtime_status_update
        # so try and log again!
        gcc_mock.side_effect = lambda *args: root_logger.error("Second Error")
        try:
            root_logger.error("First Error")
        except RuntimeError:
            self.fail("RuntimeStatusLoggingHandler should not be called recursively")


    # Test to make sure that an exception raised from
    # get_current_container doesn't cause the logger to raise an
    # exception
    @mock.patch("arvados_cwl.util.get_current_container")
    def test_runtime_status_get_current_container_exception(self, gcc_mock):
        self.setup_and_test_container_executor_and_logging(gcc_mock)
        root_logger = logging.getLogger('')

        # get_current_container is invoked when we call
        # runtime_status_update, it is going to also raise an
        # exception.
        gcc_mock.side_effect = Exception("Second Error")
        try:
            root_logger.error("First Error")
        except Exception:
            self.fail("Exception in logger should not propagate")
        self.assertTrue(gcc_mock.called)

    @mock.patch("arvados_cwl.ArvCwlExecutor.runtime_status_update")
    @mock.patch("arvados_cwl.util.get_current_container")
    @mock.patch("arvados.collection.CollectionReader")
    @mock.patch("arvados.collection.Collection")
    def test_child_failure(self, col, reader, gcc_mock, rts_mock):
        runner = self.setup_and_test_container_executor_and_logging(gcc_mock)

        gcc_mock.return_value = {"uuid" : "zzzzz-dz642-zzzzzzzzzzzzzzz"}
        self.assertTrue(gcc_mock.called)

        runner.num_retries = 0
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0
        runner.secret_store = cwltool.secrets.SecretStore()
        runner.label = mock.MagicMock()
        runner.label.return_value = '[container testjob]'

        runner.api.containers().get().execute.return_value = {
            "state":"Complete",
            "output": "abc+123",
            "exit_code": 1,
            "log": "def+234"
        }

        col().open.return_value = []

        loadingContext, runtimeContext = self.helper(runner)

        arvjob = arvados_cwl.ArvadosContainer(runner,
                                              runtimeContext,
                                              mock.MagicMock(),
                                              {},
                                              None,
                                              [],
                                              [],
                                              "testjob")
        arvjob.output_callback = mock.MagicMock()
        arvjob.collect_outputs = mock.MagicMock()
        arvjob.successCodes = [0]
        arvjob.outdir = "/var/spool/cwl"
        arvjob.output_ttl = 3600
        arvjob.collect_outputs.return_value = {"out": "stuff"}

        arvjob.done({
            "state": "Final",
            "log_uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz1",
            "output_uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2",
            "uuid": "zzzzz-xvhdp-zzzzzzzzzzzzzzz",
            "container_uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz",
            "modified_at": "2017-05-26T12:01:22Z"
        })

        rts_mock.assert_called_with(
            'error',
            'arvados.cwl-runner: [container testjob] (zzzzz-xvhdp-zzzzzzzzzzzzzzz) error log:',
            '  ** log is empty **'
        )
        arvjob.output_callback.assert_called_with({"out": "stuff"}, "permanentFail")

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_mounts(self, keepdocker):
        runner = mock.MagicMock()
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0
        runner.secret_store = cwltool.secrets.SecretStore()

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999994+99",
            "manifest_text": ". 99999999999999999999999999999994+99 0:0:file1 0:0:file2"}

        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.1")

        tool = cmap({
            "inputs": [
                {"id": "p1",
                 "type": "Directory"}
            ],
            "outputs": [],
            "baseCommand": "ls",
            "arguments": [{"valueFrom": "$(runtime.outdir)"}],
            "id": "#",
            "class": "CommandLineTool"
        })

        loadingContext, runtimeContext = self.helper(runner)
        runtimeContext.name = "test_run_mounts"

        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, loadingContext)
        arvtool.formatgraph = None
        job_order = {
            "p1": {
                "class": "Directory",
                "location": "keep:99999999999999999999999999999994+44",
                "http://arvados.org/cwl#collectionUUID": "zzzzz-4zz18-zzzzzzzzzzzzzzz",
                "listing": [
                    {
                        "class": "File",
                        "location": "keep:99999999999999999999999999999994+44/file1",
                    },
                    {
                        "class": "File",
                        "location": "keep:99999999999999999999999999999994+44/file2",
                    }
                ]
            }
        }
        for j in arvtool.job(job_order, mock.MagicMock(), runtimeContext):
            j.run(runtimeContext)
            runner.api.container_requests().create.assert_called_with(
                body=JsonDiffMatcher({
                    'environment': {
                        'HOME': '/var/spool/cwl',
                        'TMPDIR': '/tmp'
                    },
                    'name': 'test_run_mounts',
                    'runtime_constraints': {
                        'vcpus': 1,
                        'ram': 1073741824
                    },
                    'use_existing': True,
                    'priority': 500,
                    'mounts': {
                        "/keep/99999999999999999999999999999994+44": {
                            "kind": "collection",
                            "portable_data_hash": "99999999999999999999999999999994+44",
                            "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzz"
                        },
                        '/tmp': {'kind': 'tmp',
                                 "capacity": 1073741824 },
                        '/var/spool/cwl': {'kind': 'tmp',
                                           "capacity": 1073741824 }
                    },
                    'state': 'Committed',
                    'output_name': 'Output for step test_run_mounts',
                    'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                    'output_path': '/var/spool/cwl',
                    'output_ttl': 0,
                    'container_image': '99999999999999999999999999999994+99',
                    'command': ['ls', '/var/spool/cwl'],
                    'cwd': '/var/spool/cwl',
                    'scheduling_parameters': {},
                    'properties': {},
                    'secret_mounts': {}
                }))

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_secrets(self, keepdocker):
        runner = mock.MagicMock()
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0
        runner.secret_store = cwltool.secrets.SecretStore()

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.1")

        tool = cmap({"arguments": ["md5sum", "example.conf"],
                     "class": "CommandLineTool",
                     "hints": [
                         {
                             "class": "http://commonwl.org/cwltool#Secrets",
                             "secrets": [
                                 "#secret_job.cwl/pw"
                             ]
                         }
                     ],
                     "id": "#secret_job.cwl",
                     "inputs": [
                         {
                             "id": "#secret_job.cwl/pw",
                             "type": "string"
                         }
                     ],
                     "outputs": [
                     ],
                     "requirements": [
                         {
                             "class": "InitialWorkDirRequirement",
                             "listing": [
                                 {
                                     "entry": "username: user\npassword: $(inputs.pw)\n",
                                     "entryname": "example.conf"
                                 }
                             ]
                         }
                     ]})

        loadingContext, runtimeContext = self.helper(runner)
        runtimeContext.name = "test_secrets"

        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, loadingContext)
        arvtool.formatgraph = None

        job_order = {"pw": "blorp"}
        runner.secret_store.store(["pw"], job_order)

        for j in arvtool.job(job_order, mock.MagicMock(), runtimeContext):
            j.run(runtimeContext)
            runner.api.container_requests().create.assert_called_with(
                body=JsonDiffMatcher({
                    'environment': {
                        'HOME': '/var/spool/cwl',
                        'TMPDIR': '/tmp'
                    },
                    'name': 'test_secrets',
                    'runtime_constraints': {
                        'vcpus': 1,
                        'ram': 1073741824
                    },
                    'use_existing': True,
                    'priority': 500,
                    'mounts': {
                        '/tmp': {'kind': 'tmp',
                                 "capacity": 1073741824
                             },
                        '/var/spool/cwl': {'kind': 'tmp',
                                           "capacity": 1073741824 }
                    },
                    'state': 'Committed',
                    'output_name': 'Output for step test_secrets',
                    'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                    'output_path': '/var/spool/cwl',
                    'output_ttl': 0,
                    'container_image': '99999999999999999999999999999993+99',
                    'command': ['md5sum', 'example.conf'],
                    'cwd': '/var/spool/cwl',
                    'scheduling_parameters': {},
                    'properties': {},
                    "secret_mounts": {
                        "/var/spool/cwl/example.conf": {
                            "content": "username: user\npassword: blorp\n",
                            "kind": "text"
                        }
                    }
                }))

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_timelimit(self, keepdocker):
        runner = mock.MagicMock()
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0
        runner.secret_store = cwltool.secrets.SecretStore()

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        tool = cmap({
            "inputs": [],
            "outputs": [],
            "baseCommand": "ls",
            "arguments": [{"valueFrom": "$(runtime.outdir)"}],
            "id": "#",
            "class": "CommandLineTool",
            "hints": [
                {
                    "class": "ToolTimeLimit",
                    "timelimit": 42
                }
            ]
        })

        loadingContext, runtimeContext = self.helper(runner)
        runtimeContext.name = "test_timelimit"

        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, loadingContext)
        arvtool.formatgraph = None

        for j in arvtool.job({}, mock.MagicMock(), runtimeContext):
            j.run(runtimeContext)

        _, kwargs = runner.api.container_requests().create.call_args
        self.assertEqual(42, kwargs['body']['scheduling_parameters'].get('max_run_time'))


class TestWorkflow(unittest.TestCase):
    def setUp(self):
        cwltool.process._names = set()
        arv_docker_clear_cache()

    def helper(self, runner, enable_reuse=True):
        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess,
                                         collection_cache=arvados_cwl.CollectionCache(runner.api, None, 0))

        document_loader.fetcher_constructor = functools.partial(arvados_cwl.CollectionFetcher, api_client=runner.api, fs_access=make_fs_access(""))
        document_loader.fetcher = document_loader.fetcher_constructor(document_loader.cache, document_loader.session)
        document_loader.fetch_text = document_loader.fetcher.fetch_text
        document_loader.check_exists = document_loader.fetcher.check_exists

        loadingContext = arvados_cwl.context.ArvLoadingContext(
            {"avsc_names": avsc_names,
             "basedir": "",
             "make_fs_access": make_fs_access,
             "loader": document_loader,
             "metadata": {"cwlVersion": INTERNAL_VERSION, "http://commonwl.org/cwltool#original_cwlVersion": "v1.0"},
             "construct_tool_object": runner.arv_make_tool})
        runtimeContext = arvados_cwl.context.ArvRuntimeContext(
            {"work_api": "containers",
             "basedir": "",
             "name": "test_run_wf_"+str(enable_reuse),
             "make_fs_access": make_fs_access,
             "tmpdir": "/tmp",
             "enable_reuse": enable_reuse,
             "priority": 500})

        return loadingContext, runtimeContext

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.collection.CollectionReader")
    @mock.patch("arvados.collection.Collection")
    @mock.patch('arvados.commands.keepdocker.list_images_in_arv')
    def test_run(self, list_images_in_arv, mockcollection, mockcollectionreader):
        arvados_cwl.add_arv_hints()

        api = mock.MagicMock()
        api._rootDesc = get_rootDesc()

        runner = arvados_cwl.executor.ArvCwlExecutor(api)
        self.assertEqual(runner.work_api, 'containers')

        list_images_in_arv.return_value = [["zzzzz-4zz18-zzzzzzzzzzzzzzz"]]
        runner.api.collections().get().execute.return_value = {"portable_data_hash": "99999999999999999999999999999993+99"}
        runner.api.collections().list().execute.return_value = {"items": [{"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzz",
                                                                           "portable_data_hash": "99999999999999999999999999999993+99"}]}

        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.num_retries = 0
        runner.secret_store = cwltool.secrets.SecretStore()

        loadingContext, runtimeContext = self.helper(runner)
        runner.fs_access = runtimeContext.make_fs_access(runtimeContext.basedir)

        mockcollectionreader().exists.return_value = True

        tool, metadata = loadingContext.loader.resolve_ref("tests/wf/scatter2.cwl")
        metadata["cwlVersion"] = tool["cwlVersion"]

        mockc = mock.MagicMock()
        mockcollection.side_effect = lambda *args, **kwargs: CollectionMock(mockc, *args, **kwargs)
        mockcollectionreader().find.return_value = arvados.arvfile.ArvadosFile(mock.MagicMock(), "token.txt")

        arvtool = arvados_cwl.ArvadosWorkflow(runner, tool, loadingContext)
        arvtool.formatgraph = None
        it = arvtool.job({}, mock.MagicMock(), runtimeContext)

        next(it).run(runtimeContext)
        next(it).run(runtimeContext)

        with open("tests/wf/scatter2_subwf.cwl") as f:
            subwf = StripYAMLComments(f.read()).rstrip()

        runner.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher({
                "command": [
                    "cwltool",
                    "--no-container",
                    "--move-outputs",
                    "--preserve-entire-environment",
                    "workflow.cwl",
                    "cwl.input.yml"
                ],
                "container_image": "99999999999999999999999999999993+99",
                "cwd": "/var/spool/cwl",
                "environment": {
                    "HOME": "/var/spool/cwl",
                    "TMPDIR": "/tmp"
                },
                "mounts": {
                    "/keep/99999999999999999999999999999999+118": {
                        "kind": "collection",
                        "portable_data_hash": "99999999999999999999999999999999+118"
                    },
                    "/tmp": {
                        "capacity": 1073741824,
                        "kind": "tmp"
                    },
                    "/var/spool/cwl": {
                        "capacity": 1073741824,
                        "kind": "tmp"
                    },
                    "/var/spool/cwl/cwl.input.yml": {
                        "kind": "collection",
                        "path": "cwl.input.yml",
                        "portable_data_hash": "99999999999999999999999999999996+99"
                    },
                    "/var/spool/cwl/workflow.cwl": {
                        "kind": "collection",
                        "path": "workflow.cwl",
                        "portable_data_hash": "99999999999999999999999999999996+99"
                    },
                    "stdout": {
                        "kind": "file",
                        "path": "/var/spool/cwl/cwl.output.json"
                    }
                },
                "name": "scatterstep",
                "output_name": "Output for step scatterstep",
                "output_path": "/var/spool/cwl",
                "output_ttl": 0,
                "priority": 500,
                "properties": {},
                "runtime_constraints": {
                    "ram": 1073741824,
                    "vcpus": 1
                },
                "scheduling_parameters": {},
                "secret_mounts": {},
                "state": "Committed",
                "use_existing": True
            }))
        mockc.open().__enter__().write.assert_has_calls([mock.call(subwf)])
        mockc.open().__enter__().write.assert_has_calls([mock.call(
'''{
  "fileblub": {
    "basename": "token.txt",
    "class": "File",
    "location": "/keep/99999999999999999999999999999999+118/token.txt",
    "size": 0
  },
  "sleeptime": 5
}''')])

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.collection.CollectionReader")
    @mock.patch("arvados.collection.Collection")
    @mock.patch('arvados.commands.keepdocker.list_images_in_arv')
    def test_overall_resource_singlecontainer(self, list_images_in_arv, mockcollection, mockcollectionreader):
        arvados_cwl.add_arv_hints()

        api = mock.MagicMock()
        api._rootDesc = get_rootDesc()

        runner = arvados_cwl.executor.ArvCwlExecutor(api)
        self.assertEqual(runner.work_api, 'containers')

        list_images_in_arv.return_value = [["zzzzz-4zz18-zzzzzzzzzzzzzzz"]]
        runner.api.collections().get().execute.return_value = {"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzz",
                                                               "portable_data_hash": "99999999999999999999999999999993+99"}
        runner.api.collections().list().execute.return_value = {"items": [{"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzz",
                                                                           "portable_data_hash": "99999999999999999999999999999993+99"}]}

        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.num_retries = 0
        runner.secret_store = cwltool.secrets.SecretStore()

        loadingContext, runtimeContext = self.helper(runner)
        runner.fs_access = runtimeContext.make_fs_access(runtimeContext.basedir)
        loadingContext.do_update = True
        tool, metadata = loadingContext.loader.resolve_ref("tests/wf/echo-wf.cwl")

        mockcollection.side_effect = lambda *args, **kwargs: CollectionMock(mock.MagicMock(), *args, **kwargs)

        arvtool = arvados_cwl.ArvadosWorkflow(runner, tool, loadingContext)
        arvtool.formatgraph = None
        it = arvtool.job({}, mock.MagicMock(), runtimeContext)

        next(it).run(runtimeContext)
        next(it).run(runtimeContext)

        with open("tests/wf/echo-subwf.cwl") as f:
            subwf = StripYAMLComments(f.read())

        runner.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher({
                'output_ttl': 0,
                'environment': {'HOME': '/var/spool/cwl', 'TMPDIR': '/tmp'},
                'scheduling_parameters': {},
                'name': u'echo-subwf',
                'secret_mounts': {},
                'runtime_constraints': {'API': True, 'vcpus': 3, 'ram': 1073741824},
                'properties': {},
                'priority': 500,
                'mounts': {
                    '/var/spool/cwl/cwl.input.yml': {
                        'portable_data_hash': '99999999999999999999999999999996+99',
                        'kind': 'collection',
                        'path': 'cwl.input.yml'
                    },
                    '/var/spool/cwl/workflow.cwl': {
                        'portable_data_hash': '99999999999999999999999999999996+99',
                        'kind': 'collection',
                        'path': 'workflow.cwl'
                    },
                    'stdout': {
                        'path': '/var/spool/cwl/cwl.output.json',
                        'kind': 'file'
                    },
                    '/tmp': {
                        'kind': 'tmp',
                        'capacity': 1073741824
                    }, '/var/spool/cwl': {
                        'kind': 'tmp',
                        'capacity': 3221225472
                    }
                },
                'state': 'Committed',
                'output_path': '/var/spool/cwl',
                'container_image': '99999999999999999999999999999993+99',
                'command': [
                    u'cwltool',
                    u'--no-container',
                    u'--move-outputs',
                    u'--preserve-entire-environment',
                    u'workflow.cwl',
                    u'cwl.input.yml'
                ],
                'use_existing': True,
                'output_name': u'Output for step echo-subwf',
                'cwd': '/var/spool/cwl'
            }))

    def test_default_work_api(self):
        arvados_cwl.add_arv_hints()

        api = mock.MagicMock()
        api._rootDesc = copy.deepcopy(get_rootDesc())
        runner = arvados_cwl.executor.ArvCwlExecutor(api)
        self.assertEqual(runner.work_api, 'containers')
