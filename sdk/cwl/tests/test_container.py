# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados_cwl
from arvados_cwl.arvdocker import arv_docker_clear_cache
import logging
import mock
import unittest
import os
import functools
import cwltool.process
from schema_salad.ref_resolver import Loader
from schema_salad.sourceline import cmap

from .matcher import JsonDiffMatcher

if not os.getenv('ARVADOS_DEBUG'):
    logging.getLogger('arvados.cwl-runner').setLevel(logging.WARN)
    logging.getLogger('arvados.arv-run').setLevel(logging.WARN)


class TestContainer(unittest.TestCase):

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_run(self, keepdocker):
        for enable_reuse in (True, False):
            arv_docker_clear_cache()

            runner = mock.MagicMock()
            runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
            runner.ignore_docker_for_reuse = False
            runner.intermediate_output_ttl = 0

            keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
            runner.api.collections().get().execute.return_value = {
                "portable_data_hash": "99999999999999999999999999999993+99"}

            document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

            tool = cmap({
                "inputs": [],
                "outputs": [],
                "baseCommand": "ls",
                "arguments": [{"valueFrom": "$(runtime.outdir)"}],
                "id": "#",
                "class": "CommandLineTool"
            })
            make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess,
                                         collection_cache=arvados_cwl.CollectionCache(runner.api, None, 0))
            arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, work_api="containers", avsc_names=avsc_names,
                                                     basedir="", make_fs_access=make_fs_access, loader=Loader({}))
            arvtool.formatgraph = None
            for j in arvtool.job({}, mock.MagicMock(), basedir="", name="test_run_"+str(enable_reuse),
                                 make_fs_access=make_fs_access, tmpdir="/tmp"):
                j.run(enable_reuse=enable_reuse)
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
                        'priority': 1,
                        'mounts': {
                            '/tmp': {'kind': 'tmp',
                                     "capacity": 1073741824
                                 },
                            '/var/spool/cwl': {'kind': 'tmp',
                                               "capacity": 1073741824 }
                        },
                        'state': 'Committed',
                        'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                        'output_path': '/var/spool/cwl',
                        'output_ttl': 0,
                        'container_image': 'arvados/jobs',
                        'command': ['ls', '/var/spool/cwl'],
                        'cwd': '/var/spool/cwl',
                        'scheduling_parameters': {},
                        'properties': {},
                    }))

    # The test passes some fields in builder.resources
    # For the remaining fields, the defaults will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_resource_requirements(self, keepdocker):
        arv_docker_clear_cache()
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 3600
        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

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
        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess,
                                         collection_cache=arvados_cwl.CollectionCache(runner.api, None, 0))
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, work_api="containers",
                                                 avsc_names=avsc_names, make_fs_access=make_fs_access,
                                                 loader=Loader({}))
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), basedir="", name="test_resource_requirements",
                             make_fs_access=make_fs_access, tmpdir="/tmp"):
            j.run(enable_reuse=True)

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
            'priority': 1,
            'mounts': {
                '/tmp': {'kind': 'tmp',
                         "capacity": 4194304000 },
                '/var/spool/cwl': {'kind': 'tmp',
                                   "capacity": 5242880000 }
            },
            'state': 'Committed',
            'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
            'output_path': '/var/spool/cwl',
            'output_ttl': 7200,
            'container_image': 'arvados/jobs',
            'command': ['ls'],
            'cwd': '/var/spool/cwl',
            'scheduling_parameters': {
                'partitions': ['blurb']
            },
            'properties': {}
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
        arv_docker_clear_cache()
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0
        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

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
        collection_mock.return_value = vwdmock
        vwdmock.portable_data_hash.return_value = "99999999999999999999999999999996+99"

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
        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess,
                                         collection_cache=arvados_cwl.CollectionCache(runner.api, None, 0))
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, work_api="containers",
                                                 avsc_names=avsc_names, make_fs_access=make_fs_access,
                                                 loader=Loader({}))
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), basedir="", name="test_initial_work_dir",
                             make_fs_access=make_fs_access, tmpdir="/tmp"):
            j.run()

        call_args, call_kwargs = runner.api.container_requests().create.call_args

        vwdmock.copy.assert_has_calls([mock.call('bar', 'foo', source_collection=sourcemock)])
        vwdmock.copy.assert_has_calls([mock.call('', 'foo2', source_collection=sourcemock)])
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
            'priority': 1,
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
            'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
            'output_path': '/var/spool/cwl',
            'output_ttl': 0,
            'container_image': 'arvados/jobs',
            'command': ['ls'],
            'cwd': '/var/spool/cwl',
            'scheduling_parameters': {
            },
            'properties': {}
        }

        call_body = call_kwargs.get('body', None)
        self.assertNotEqual(None, call_body)
        for key in call_body:
            self.assertEqual(call_body_expected.get(key), call_body.get(key))


    # Test redirecting stdin/stdout/stderr
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_redirects(self, keepdocker):
        arv_docker_clear_cache()

        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

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
        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess,
                                         collection_cache=arvados_cwl.CollectionCache(runner.api, None, 0))
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, work_api="containers", avsc_names=avsc_names,
                                                 basedir="", make_fs_access=make_fs_access, loader=Loader({}))
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), basedir="", name="test_run_redirect",
                             make_fs_access=make_fs_access, tmpdir="/tmp"):
            j.run()
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
                    'priority': 1,
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
                    'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                    'output_path': '/var/spool/cwl',
                    'output_ttl': 0,
                    'container_image': 'arvados/jobs',
                    'command': ['ls', '/var/spool/cwl'],
                    'cwd': '/var/spool/cwl',
                    'scheduling_parameters': {},
                    'properties': {},
                }))

    @mock.patch("arvados.collection.Collection")
    def test_done(self, col):
        api = mock.MagicMock()

        runner = mock.MagicMock()
        runner.api = api
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.num_retries = 0
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0

        runner.api.containers().get().execute.return_value = {"state":"Complete",
                                                              "output": "abc+123",
                                                              "exit_code": 0}

        col().open.return_value = []

        arvjob = arvados_cwl.ArvadosContainer(runner)
        arvjob.name = "testjob"
        arvjob.builder = mock.MagicMock()
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

        arvjob.collect_outputs.assert_called_with("keep:abc+123")
        arvjob.output_callback.assert_called_with({"out": "stuff"}, "success")
        runner.add_intermediate_output.assert_called_with("zzzzz-4zz18-zzzzzzzzzzzzzz2")

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_mounts(self, keepdocker):
        arv_docker_clear_cache()

        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.intermediate_output_ttl = 0

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

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
        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess,
                                     collection_cache=arvados_cwl.CollectionCache(runner.api, None, 0))
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, work_api="containers", avsc_names=avsc_names,
                                                 basedir="", make_fs_access=make_fs_access, loader=Loader({}))
        arvtool.formatgraph = None
        job_order = {
            "p1": {
                "class": "Directory",
                "location": "keep:99999999999999999999999999999994+44",
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
        for j in arvtool.job(job_order, mock.MagicMock(), basedir="", name="test_run_mounts",
                             make_fs_access=make_fs_access, tmpdir="/tmp"):
            j.run()
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
                    'priority': 1,
                    'mounts': {
                        "/keep/99999999999999999999999999999994+44": {
                            "kind": "collection",
                            "portable_data_hash": "99999999999999999999999999999994+44"
                        },
                        '/tmp': {'kind': 'tmp',
                                 "capacity": 1073741824 },
                        '/var/spool/cwl': {'kind': 'tmp',
                                           "capacity": 1073741824 }
                    },
                    'state': 'Committed',
                    'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                    'output_path': '/var/spool/cwl',
                    'output_ttl': 0,
                    'container_image': 'arvados/jobs',
                    'command': ['ls', '/var/spool/cwl'],
                    'cwd': '/var/spool/cwl',
                    'scheduling_parameters': {},
                    'properties': {},
                }))
