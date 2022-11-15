# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from future import standard_library
standard_library.install_aliases()
from builtins import object
from builtins import str
from future.utils import viewvalues

import copy
import io
import functools
import hashlib
import json
import logging
import mock
import sys
import unittest
import cwltool.process
import re
import os

from io import BytesIO

# StringIO.StringIO and io.StringIO have different behavior write() is
# called with both python2 (byte) strings and unicode strings
# (specifically there's some logging in cwltool that causes trouble).
# This isn't a problem on python3 because all string are unicode.
if sys.version_info[0] < 3:
    from StringIO import StringIO
else:
    from io import StringIO

import arvados
import arvados.collection
import arvados_cwl
import arvados_cwl.executor
import arvados_cwl.runner
import arvados.keep

from .matcher import JsonDiffMatcher, StripYAMLComments
from .mock_discovery import get_rootDesc

import ruamel.yaml as yaml

_rootDesc = None

def stubs(wfdetails=('submit_wf.cwl', None)):
    def outer_wrapper(func, *rest):
        @functools.wraps(func)
        @mock.patch("arvados_cwl.arvdocker.determine_image_id")
        @mock.patch("uuid.uuid4")
        @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
        @mock.patch("arvados.collection.KeepClient")
        @mock.patch("arvados.keep.KeepClient")
        @mock.patch("arvados.events.subscribe")
        def wrapped(self, events, keep_client1, keep_client2, keepdocker,
                    uuid4, determine_image_id, *args, **kwargs):
            class Stubs(object):
                pass

            wfname = wfdetails[0]
            wfpath = wfdetails[1]

            stubs = Stubs()
            stubs.events = events
            stubs.keepdocker = keepdocker

            uuid4.side_effect = ["df80736f-f14d-4b10-b2e3-03aa27f034bb", "df80736f-f14d-4b10-b2e3-03aa27f034b1",
                                 "df80736f-f14d-4b10-b2e3-03aa27f034b2", "df80736f-f14d-4b10-b2e3-03aa27f034b3",
                                 "df80736f-f14d-4b10-b2e3-03aa27f034b4", "df80736f-f14d-4b10-b2e3-03aa27f034b5"]

            determine_image_id.return_value = None

            def putstub(p, **kwargs):
                return "%s+%i" % (hashlib.md5(p).hexdigest(), len(p))
            keep_client1().put.side_effect = putstub
            keep_client1.put.side_effect = putstub
            keep_client2().put.side_effect = putstub
            keep_client2.put.side_effect = putstub

            stubs.keep_client = keep_client2
            stubs.docker_images = {
                "arvados/jobs:"+arvados_cwl.__version__: [("zzzzz-4zz18-zzzzzzzzzzzzzd3", {})],
                "debian:buster-slim": [("zzzzz-4zz18-zzzzzzzzzzzzzd4", {})],
                "arvados/jobs:123": [("zzzzz-4zz18-zzzzzzzzzzzzzd5", {})],
                "arvados/jobs:latest": [("zzzzz-4zz18-zzzzzzzzzzzzzd6", {})],
            }
            def kd(a, b, image_name=None, image_tag=None, project_uuid=None):
                return stubs.docker_images.get("%s:%s" % (image_name, image_tag), [])
            stubs.keepdocker.side_effect = kd

            stubs.fake_user_uuid = "zzzzz-tpzed-zzzzzzzzzzzzzzz"
            stubs.fake_container_uuid = "zzzzz-dz642-zzzzzzzzzzzzzzz"

            if sys.version_info[0] < 3:
                stubs.capture_stdout = BytesIO()
            else:
                stubs.capture_stdout = StringIO()

            stubs.api = mock.MagicMock()
            stubs.api._rootDesc = get_rootDesc()
            stubs.api._rootDesc["uuidPrefix"] = "zzzzz"
            stubs.api._rootDesc["revision"] = "20210628"

            stubs.api.users().current().execute.return_value = {
                "uuid": stubs.fake_user_uuid,
            }
            stubs.api.collections().list().execute.return_value = {"items": []}
            stubs.api.containers().current().execute.return_value = {
                "uuid": stubs.fake_container_uuid,
            }
            stubs.api.config()["StorageClasses"].items.return_value = {
                "default": {
                    "Default": True
                }
            }.items()

            class CollectionExecute(object):
                def __init__(self, exe):
                    self.exe = exe
                def execute(self, num_retries=None):
                    return self.exe

            def collection_createstub(created_collections, body, ensure_unique_name=None):
                mt = body["manifest_text"].encode('utf-8')
                uuid = "zzzzz-4zz18-zzzzzzzzzzzzzx%d" % len(created_collections)
                pdh = "%s+%i" % (hashlib.md5(mt).hexdigest(), len(mt))
                created_collections[uuid] = {
                    "uuid": uuid,
                    "portable_data_hash": pdh,
                    "manifest_text": mt.decode('utf-8')
                }
                return CollectionExecute(created_collections[uuid])

            def collection_getstub(created_collections, uuid):
                for v in viewvalues(created_collections):
                    if uuid in (v["uuid"], v["portable_data_hash"]):
                        return CollectionExecute(v)

            created_collections = {
                "99999999999999999999999999999998+99": {
                    "uuid": "",
                    "portable_data_hash": "99999999999999999999999999999998+99",
                    "manifest_text": ". 99999999999999999999999999999998+99 0:0:file1.txt"
                },
                "99999999999999999999999999999997+99": {
                    "uuid": "",
                    "portable_data_hash": "99999999999999999999999999999997+99",
                    "manifest_text": ". 99999999999999999999999999999997+99 0:0:file1.txt"
                },
                "99999999999999999999999999999994+99": {
                    "uuid": "",
                    "portable_data_hash": "99999999999999999999999999999994+99",
                    "manifest_text": ". 99999999999999999999999999999994+99 0:0:expect_arvworkflow.cwl"
                },
                "zzzzz-4zz18-zzzzzzzzzzzzzd3": {
                    "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzd3",
                    "portable_data_hash": "999999999999999999999999999999d3+99",
                    "manifest_text": ""
                },
                "zzzzz-4zz18-zzzzzzzzzzzzzd4": {
                    "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzd4",
                    "portable_data_hash": "999999999999999999999999999999d4+99",
                    "manifest_text": ""
                },
                "zzzzz-4zz18-zzzzzzzzzzzzzd5": {
                    "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzd5",
                    "portable_data_hash": "999999999999999999999999999999d5+99",
                    "manifest_text": ""
                },
                "zzzzz-4zz18-zzzzzzzzzzzzzd6": {
                    "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzd6",
                    "portable_data_hash": "999999999999999999999999999999d6+99",
                    "manifest_text": ""
                }
            }
            stubs.api.collections().create.side_effect = functools.partial(collection_createstub, created_collections)
            stubs.api.collections().get.side_effect = functools.partial(collection_getstub, created_collections)

            stubs.expect_job_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
            stubs.api.jobs().create().execute.return_value = {
                "uuid": stubs.expect_job_uuid,
                "state": "Queued",
            }

            stubs.expect_container_request_uuid = "zzzzz-xvhdp-zzzzzzzzzzzzzzz"
            stubs.api.container_requests().create().execute.return_value = {
                "uuid": stubs.expect_container_request_uuid,
                "container_uuid": "zzzzz-dz642-zzzzzzzzzzzzzzz",
                "state": "Queued"
            }

            stubs.expect_pipeline_template_uuid = "zzzzz-d1hrv-zzzzzzzzzzzzzzz"
            stubs.api.pipeline_templates().create().execute.return_value = {
                "uuid": stubs.expect_pipeline_template_uuid,
            }
            stubs.expect_job_spec = {
                'runtime_constraints': {
                    'docker_image': '999999999999999999999999999999d3+99',
                    'min_ram_mb_per_node': 1024
                },
                'script_parameters': {
                    'x': {
                        'basename': 'blorp.txt',
                        'location': 'keep:169f39d466a5438ac4a90e779bf750c7+53/blorp.txt',
                        'class': 'File'
                    },
                    'y': {
                        'basename': '99999999999999999999999999999998+99',
                        'location': 'keep:99999999999999999999999999999998+99',
                        'class': 'Directory'
                    },
                    'z': {
                        'basename': 'anonymous',
                        "listing": [{
                            "basename": "renamed.txt",
                            "class": "File",
                            "location": "keep:99999999999999999999999999999998+99/file1.txt",
                            "size": 0
                        }],
                        'class': 'Directory'
                    },
                    'cwl:tool': '57ad063d64c60dbddc027791f0649211+60/workflow.cwl#main'
                },
                'repository': 'arvados',
                'script_version': 'master',
                'minimum_script_version': '570509ab4d2ef93d870fd2b1f2eab178afb1bad9',
                'script': 'cwl-runner'
            }
            stubs.pipeline_component = stubs.expect_job_spec.copy()
            stubs.expect_pipeline_instance = {
                'name': 'submit_wf.cwl',
                'state': 'RunningOnServer',
                'owner_uuid': None,
                "components": {
                    "cwl-runner": {
                        'runtime_constraints': {'docker_image': '999999999999999999999999999999d3+99', 'min_ram_mb_per_node': 1024},
                        'script_parameters': {
                            'y': {"value": {'basename': '99999999999999999999999999999998+99', 'location': 'keep:99999999999999999999999999999998+99', 'class': 'Directory'}},
                            'x': {"value": {
                                'basename': 'blorp.txt',
                                'class': 'File',
                                'location': 'keep:169f39d466a5438ac4a90e779bf750c7+53/blorp.txt',
                                "size": 16
                            }},
                            'z': {"value": {'basename': 'anonymous', 'class': 'Directory',
                                  'listing': [
                                      {
                                          'basename': 'renamed.txt',
                                          'class': 'File', 'location':
                                          'keep:99999999999999999999999999999998+99/file1.txt',
                                          'size': 0
                                      }
                                  ]}},
                            'cwl:tool': '57ad063d64c60dbddc027791f0649211+60/workflow.cwl#main',
                            'arv:debug': True,
                            'arv:enable_reuse': True,
                            'arv:on_error': 'continue'
                        },
                        'repository': 'arvados',
                        'script_version': 'master',
                        'minimum_script_version': '570509ab4d2ef93d870fd2b1f2eab178afb1bad9',
                        'script': 'cwl-runner',
                        'job': {'state': 'Queued', 'uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'}
                    }
                }
            }
            stubs.pipeline_create = copy.deepcopy(stubs.expect_pipeline_instance)
            stubs.expect_pipeline_uuid = "zzzzz-d1hrv-zzzzzzzzzzzzzzz"
            stubs.pipeline_create["uuid"] = stubs.expect_pipeline_uuid
            stubs.pipeline_with_job = copy.deepcopy(stubs.pipeline_create)
            stubs.pipeline_with_job["components"]["cwl-runner"]["job"] = {
                "uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz",
                "state": "Queued"
            }
            stubs.api.pipeline_instances().create().execute.return_value = stubs.pipeline_create
            stubs.api.pipeline_instances().get().execute.return_value = stubs.pipeline_with_job

            cwd = os.getcwd()
            filepath = os.path.join(cwd, "tests/wf/submit_wf_packed.cwl")
            with open(filepath) as f:
                expect_packed_workflow = yaml.round_trip_load(f)

            if wfpath is None:
                wfpath = wfname

            gitinfo_workflow = copy.deepcopy(expect_packed_workflow)
            gitinfo_workflow["$graph"][0]["id"] = "file://%s/tests/wf/%s" % (cwd, wfpath)
            mocktool = mock.NonCallableMock(tool=gitinfo_workflow["$graph"][0], metadata=gitinfo_workflow)

            stubs.git_info = arvados_cwl.executor.ArvCwlExecutor.get_git_info(mocktool)
            expect_packed_workflow.update(stubs.git_info)

            stubs.git_props = {"arv:"+k.split("#", 1)[1]: v for k,v in stubs.git_info.items()}

            if wfname == wfpath:
                container_name = "%s (%s)" % (wfpath, stubs.git_props["arv:gitDescribe"])
            else:
                container_name = wfname

            stubs.expect_container_spec = {
                'priority': 500,
                'mounts': {
                    '/var/spool/cwl': {
                        'writable': True,
                        'kind': 'collection'
                    },
                    '/var/lib/cwl/workflow.json': {
                        'content': expect_packed_workflow,
                        'kind': 'json'
                    },
                    'stdout': {
                        'path': '/var/spool/cwl/cwl.output.json',
                        'kind': 'file'
                    },
                    '/var/lib/cwl/cwl.input.json': {
                        'kind': 'json',
                        'content': {
                            'y': {
                                'basename': '99999999999999999999999999999998+99',
                                'location': 'keep:99999999999999999999999999999998+99',
                                'class': 'Directory'},
                            'x': {
                                'basename': u'blorp.txt',
                                'class': 'File',
                                'location': u'keep:169f39d466a5438ac4a90e779bf750c7+53/blorp.txt',
                                "size": 16
                            },
                            'z': {'basename': 'anonymous', 'class': 'Directory', 'listing': [
                                {'basename': 'renamed.txt',
                                 'class': 'File',
                                 'location': 'keep:99999999999999999999999999999998+99/file1.txt',
                                 'size': 0
                                }
                            ]}
                        },
                        'kind': 'json'
                    }
                },
                'secret_mounts': {},
                'state': 'Committed',
                'command': ['arvados-cwl-runner', '--local', '--api=containers',
                            '--no-log-timestamps', '--disable-validate', '--disable-color',
                            '--eval-timeout=20', '--thread-count=0',
                            '--enable-reuse', "--collection-cache-size=256",
                            '--output-name=Output from workflow '+container_name,
                            '--debug', '--on-error=continue',
                            '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json'],
                'name': container_name,
                'container_image': '999999999999999999999999999999d3+99',
                'output_name': 'Output from workflow %s' % (container_name),
                'output_path': '/var/spool/cwl',
                'cwd': '/var/spool/cwl',
                'runtime_constraints': {
                    'API': True,
                    'vcpus': 1,
                    'ram': (1024+256)*1024*1024
                },
                'use_existing': False,
                'properties': stubs.git_props,
                'secret_mounts': {}
            }

            stubs.expect_workflow_uuid = "zzzzz-7fd4e-zzzzzzzzzzzzzzz"
            stubs.api.workflows().create().execute.return_value = {
                "uuid": stubs.expect_workflow_uuid,
            }
            def update_mock(**kwargs):
                stubs.updated_uuid = kwargs.get('uuid')
                return mock.DEFAULT
            stubs.api.workflows().update.side_effect = update_mock
            stubs.api.workflows().update().execute.side_effect = lambda **kwargs: {
                "uuid": stubs.updated_uuid,
            }

            return func(self, stubs, *args, **kwargs)
        return wrapped
    return outer_wrapper

class TestSubmit(unittest.TestCase):

    def setUp(self):
        cwltool.process._names = set()
        arvados_cwl.arvdocker.arv_docker_clear_cache()

    def tearDown(self):
        root_logger = logging.getLogger('')

        # Remove existing RuntimeStatusLoggingHandlers if they exist
        handlers = [h for h in root_logger.handlers if not isinstance(h, arvados_cwl.executor.RuntimeStatusLoggingHandler)]
        root_logger.handlers = handlers

    @mock.patch("time.sleep")
    @stubs()
    def test_submit_invalid_runner_ram(self, stubs, tm):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--debug", "--submit-runner-ram=-2048",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)
        self.assertEqual(exited, 1)


    @stubs()
    def test_submit_container(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        stubs.api.collections().create.assert_has_calls([
            mock.call(body=JsonDiffMatcher({
                'manifest_text':
                '. 979af1245a12a1fed634d4222473bfdc+16 0:16:blorp.txt\n',
                'replication_desired': None,
                'name': 'submit_wf.cwl ('+ stubs.git_props["arv:gitDescribe"] +') input (169f39d466a5438ac4a90e779bf750c7+53)',
            }), ensure_unique_name=False),
            mock.call(body=JsonDiffMatcher({
                'manifest_text':
                '. 5bcc9fe8f8d5992e6cf418dc7ce4dbb3+16 0:16:blub.txt\n',
                'replication_desired': None,
                'name': 'submit_tool.cwl dependencies (5d373e7629203ce39e7c22af98a0f881+52)',
            }), ensure_unique_name=False),
            ])

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)


    @stubs()
    def test_submit_container_tool(self, stubs):
        # test for issue #16139
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
                "tests/tool/tool_with_sf.cwl", "tests/tool/tool_with_sf.yml"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_no_reuse(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--disable-reuse",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = [
            'arvados-cwl-runner', '--local', '--api=containers',
            '--no-log-timestamps', '--disable-validate', '--disable-color',
            '--eval-timeout=20', '--thread-count=0',
            '--disable-reuse', "--collection-cache-size=256",
            '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
            '--debug', '--on-error=continue',
            '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']
        expect_container["use_existing"] = False

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs(('submit_wf_no_reuse.cwl', None))
    def test_submit_container_reuse_disabled_by_workflow(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
             "tests/wf/submit_wf_no_reuse.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)
        self.assertEqual(exited, 0)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ["--disable-reuse" if v == "--enable-reuse" else v for v in expect_container["command"]]
        expect_container["use_existing"] = False
        expect_container["mounts"]["/var/lib/cwl/workflow.json"]["content"]["$graph"][1]["hints"] = [
            {
                "class": "http://arvados.org/cwl#ReuseRequirement",
                "enableReuse": False,
            },
        ]
        expect_container["mounts"]["/var/lib/cwl/workflow.json"]["content"]["$namespaces"] = {
            "arv": "http://arvados.org/cwl#",
            "cwltool": "http://commonwl.org/cwltool#"
        }

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')


    @stubs()
    def test_submit_container_on_error(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--on-error=stop",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', '--on-error=stop',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_output_name(self, stubs):
        output_name = "test_output_name"

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--output-name", output_name,
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       "--output-name="+output_name, '--debug', '--on-error=continue',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']
        expect_container["output_name"] = output_name

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_storage_classes(self, stubs):
        exited = arvados_cwl.main(
            ["--debug", "--submit", "--no-wait", "--api=containers", "--storage-classes=foo",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       "--debug",
                                       "--storage-classes=foo", '--on-error=continue',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_multiple_storage_classes(self, stubs):
        exited = arvados_cwl.main(
            ["--debug", "--submit", "--no-wait", "--api=containers", "--storage-classes=foo,bar", "--intermediate-storage-classes=baz",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       "--debug",
                                       "--storage-classes=foo,bar", "--intermediate-storage-classes=baz", '--on-error=continue',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @mock.patch("cwltool.task_queue.TaskQueue")
    @mock.patch("arvados_cwl.arvworkflow.ArvadosWorkflow.job")
    @mock.patch("arvados_cwl.executor.ArvCwlExecutor.make_output_collection")
    @stubs()
    def test_storage_classes_correctly_propagate_to_make_output_collection(self, stubs, make_output, job, tq):
        final_output_c = arvados.collection.Collection()
        make_output.return_value = ({},final_output_c)

        def set_final_output(job_order, output_callback, runtimeContext):
            output_callback({"out": "zzzzz"}, "success")
            return []
        job.side_effect = set_final_output

        exited = arvados_cwl.main(
            ["--debug", "--local", "--storage-classes=foo", "--disable-git",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        make_output.assert_called_with(u'Output from workflow submit_wf.cwl', ['foo'], '', {}, {"out": "zzzzz"})
        self.assertEqual(exited, 0)

    @mock.patch("cwltool.task_queue.TaskQueue")
    @mock.patch("arvados_cwl.arvworkflow.ArvadosWorkflow.job")
    @mock.patch("arvados_cwl.executor.ArvCwlExecutor.make_output_collection")
    @stubs()
    def test_default_storage_classes_correctly_propagate_to_make_output_collection(self, stubs, make_output, job, tq):
        final_output_c = arvados.collection.Collection()
        make_output.return_value = ({},final_output_c)
        stubs.api.config().get.return_value = {"default": {"Default": True}}

        def set_final_output(job_order, output_callback, runtimeContext):
            output_callback({"out": "zzzzz"}, "success")
            return []
        job.side_effect = set_final_output

        exited = arvados_cwl.main(
            ["--debug", "--local", "--disable-git",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        make_output.assert_called_with(u'Output from workflow submit_wf.cwl', ['default'], '', {}, {"out": "zzzzz"})
        self.assertEqual(exited, 0)

    @mock.patch("cwltool.task_queue.TaskQueue")
    @mock.patch("arvados_cwl.arvworkflow.ArvadosWorkflow.job")
    @mock.patch("arvados_cwl.executor.ArvCwlExecutor.make_output_collection")
    @stubs()
    def test_storage_class_hint_to_make_output_collection(self, stubs, make_output, job, tq):
        final_output_c = arvados.collection.Collection()
        make_output.return_value = ({},final_output_c)

        def set_final_output(job_order, output_callback, runtimeContext):
            output_callback({"out": "zzzzz"}, "success")
            return []
        job.side_effect = set_final_output

        exited = arvados_cwl.main(
            ["--debug", "--local", "--disable-git",
                "tests/wf/submit_storage_class_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        make_output.assert_called_with(u'Output from workflow submit_storage_class_wf.cwl', ['foo', 'bar'], '', {}, {"out": "zzzzz"})
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_output_ttl(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--intermediate-output-ttl", "3600",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug',
                                       '--on-error=continue',
                                       "--intermediate-output-ttl=3600",
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_trash_intermediate(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--trash-intermediate",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)


        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', '--on-error=continue',
                                       "--trash-intermediate",
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_output_tags(self, stubs):
        output_tags = "tag0,tag1,tag2"

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--output-tags", output_tags,
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       "--output-tags="+output_tags, '--debug', '--on-error=continue',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_runner_ram(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--submit-runner-ram=2048",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["runtime_constraints"]["ram"] = (2048+256)*1024*1024

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @mock.patch("arvados.collection.CollectionReader")
    @mock.patch("time.sleep")
    @stubs()
    def test_submit_file_keepref(self, stubs, tm, collectionReader):
        collectionReader().exists.return_value = True
        collectionReader().find.return_value = arvados.arvfile.ArvadosFile(mock.MagicMock(), "blorp.txt")
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
             "tests/wf/submit_keepref_wf.cwl"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)
        self.assertEqual(exited, 0)

    @mock.patch("arvados.collection.CollectionReader")
    @mock.patch("time.sleep")
    @stubs()
    def test_submit_keepref(self, stubs, tm, reader):
        with open("tests/wf/expect_arvworkflow.cwl") as f:
            reader().open().__enter__().read.return_value = f.read()

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
             "keep:99999999999999999999999999999994+99/expect_arvworkflow.cwl#main", "-x", "XxX"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        expect_container = {
            'priority': 500,
            'mounts': {
                '/var/spool/cwl': {
                    'writable': True,
                    'kind': 'collection'
                },
                'stdout': {
                    'path': '/var/spool/cwl/cwl.output.json',
                    'kind': 'file'
                },
                '/var/lib/cwl/workflow': {
                    'portable_data_hash': '99999999999999999999999999999994+99',
                    'kind': 'collection'
                },
                '/var/lib/cwl/cwl.input.json': {
                    'content': {
                        'x': 'XxX'
                    },
                    'kind': 'json'
                }
            }, 'state': 'Committed',
            'output_path': '/var/spool/cwl',
            'name': 'expect_arvworkflow.cwl#main',
            'output_name': 'Output from workflow expect_arvworkflow.cwl#main',
            'container_image': '999999999999999999999999999999d3+99',
            'command': ['arvados-cwl-runner', '--local', '--api=containers',
                        '--no-log-timestamps', '--disable-validate', '--disable-color',
                        '--eval-timeout=20', '--thread-count=0',
                        '--enable-reuse', "--collection-cache-size=256",
                        '--output-name=Output from workflow expect_arvworkflow.cwl#main',
                        '--debug', '--on-error=continue',
                        '/var/lib/cwl/workflow/expect_arvworkflow.cwl#main', '/var/lib/cwl/cwl.input.json'],
            'cwd': '/var/spool/cwl',
            'runtime_constraints': {
                'API': True,
                'vcpus': 1,
                'ram': 1342177280
            },
            'use_existing': False,
            'properties': {},
            'secret_mounts': {}
        }

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @mock.patch("time.sleep")
    @stubs()
    def test_submit_arvworkflow(self, stubs, tm):
        with open("tests/wf/expect_arvworkflow.cwl") as f:
            stubs.api.workflows().get().execute.return_value = {"definition": f.read(), "name": "a test workflow"}

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--disable-git",
             "962eh-7fd4e-gkbzl62qqtfig37", "-x", "XxX"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        expect_container = {
            'priority': 500,
            'mounts': {
                '/var/spool/cwl': {
                    'writable': True,
                    'kind': 'collection'
                },
                'stdout': {
                    'path': '/var/spool/cwl/cwl.output.json',
                    'kind': 'file'
                },
                '/var/lib/cwl/workflow.json': {
                    'kind': 'json',
                    'content': {
                        'cwlVersion': 'v1.0',
                        '$graph': [
                            {
                                'id': '#main',
                                'inputs': [
                                    {'type': 'string', 'id': '#main/x'}
                                ],
                                'steps': [
                                    {'in': [{'source': '#main/x', 'id': '#main/step1/x'}],
                                     'run': '#submit_tool.cwl',
                                     'id': '#main/step1',
                                     'out': []}
                                ],
                                'class': 'Workflow',
                                'outputs': []
                            },
                            {
                                'inputs': [
                                    {
                                        'inputBinding': {'position': 1},
                                        'type': 'string',
                                        'id': '#submit_tool.cwl/x'}
                                ],
                                'requirements': [
                                    {
                                        'dockerPull': 'debian:buster-slim',
                                        'class': 'DockerRequirement'
                                    }
                                ],
                                'id': '#submit_tool.cwl',
                                'outputs': [],
                                'baseCommand': 'cat',
                                'class': 'CommandLineTool'
                            }
                        ]
                    }
                },
                '/var/lib/cwl/cwl.input.json': {
                    'content': {
                        'x': 'XxX'
                    },
                    'kind': 'json'
                }
            }, 'state': 'Committed',
            'output_path': '/var/spool/cwl',
            'name': 'a test workflow',
            'container_image': "999999999999999999999999999999d3+99",
            'command': ['arvados-cwl-runner', '--local', '--api=containers',
                        '--no-log-timestamps', '--disable-validate', '--disable-color',
                        '--eval-timeout=20', '--thread-count=0',
                        '--enable-reuse', "--collection-cache-size=256",
                        "--output-name=Output from workflow a test workflow",
                        '--debug', '--on-error=continue',
                        '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json'],
            'output_name': 'Output from workflow a test workflow',
            'cwd': '/var/spool/cwl',
            'runtime_constraints': {
                'API': True,
                'vcpus': 1,
                'ram': 1342177280
            },
            'use_existing': False,
            'properties': {
                "template_uuid": "962eh-7fd4e-gkbzl62qqtfig37"
            },
            'secret_mounts': {}
        }

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs(('hello container 123', 'submit_wf.cwl'))
    def test_submit_container_name(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--name=hello container 123",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_missing_input(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)
        self.assertEqual(exited, 0)

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job_missing.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)
        self.assertEqual(exited, 1)

    @stubs()
    def test_submit_container_project(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'
        stubs.api.groups().get().execute.return_value = {"group_class": "project"}
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--project-uuid="+project_uuid,
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["owner_uuid"] = project_uuid
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       "--eval-timeout=20", "--thread-count=0",
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug',
                                       '--on-error=continue',
                                       '--project-uuid='+project_uuid,
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_eval_timeout(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--eval-timeout=60",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=60.0', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', '--on-error=continue',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_collection_cache(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--collection-cache-size=500",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=500",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', '--on-error=continue',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']
        expect_container["runtime_constraints"]["ram"] = (1024+500)*1024*1024

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_thread_count(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--thread-count=20",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=20',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', '--on-error=continue',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_runner_image(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--submit-runner-image=arvados/jobs:123",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        stubs.expect_container_spec["container_image"] = "999999999999999999999999999999d5+99"

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_priority(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--priority=669",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        stubs.expect_container_spec["priority"] = 669

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs(('submit_wf_runner_resources.cwl', None))
    def test_submit_wf_runner_resources(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
                "tests/wf/submit_wf_runner_resources.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["runtime_constraints"] = {
            "API": True,
            "vcpus": 2,
            "ram": (2000+512) * 2**20
        }
        expect_container["mounts"]["/var/lib/cwl/workflow.json"]["content"]["$graph"][1]["hints"] = [
            {
                "class": "http://arvados.org/cwl#WorkflowRunnerResources",
                "coresMin": 2,
                "ramMin": 2000,
                "keep_cache": 512
            }
        ]
        expect_container["mounts"]["/var/lib/cwl/workflow.json"]["content"]["$namespaces"] = {
            "arv": "http://arvados.org/cwl#",
        }
        expect_container["command"] = ["--collection-cache-size=512" if v == "--collection-cache-size=256" else v for v in expect_container["command"]]

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @mock.patch("arvados.commands.keepdocker.find_one_image_hash")
    @mock.patch("cwltool.docker.DockerCommandLineJob.get_image")
    @mock.patch("arvados.api")
    def test_arvados_jobs_image(self, api, get_image, find_one_image_hash):
        arvados_cwl.arvdocker.arv_docker_clear_cache()

        arvrunner = mock.MagicMock()
        arvrunner.project_uuid = ""
        api.return_value = mock.MagicMock()
        arvrunner.api = api.return_value
        arvrunner.runtimeContext.match_local_docker = False
        arvrunner.api.links().list().execute.side_effect = ({"items": [{"created_at": "",
                                                                        "head_uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzb",
                                                                        "link_class": "docker_image_repo+tag",
                                                                        "name": "arvados/jobs:"+arvados_cwl.__version__,
                                                                        "owner_uuid": "",
                                                                        "properties": {"image_timestamp": ""}}], "items_available": 1, "offset": 0},
                                                            {"items": [{"created_at": "",
                                                                        "head_uuid": "",
                                                                        "link_class": "docker_image_hash",
                                                                        "name": "123456",
                                                                        "owner_uuid": "",
                                                                        "properties": {"image_timestamp": ""}}], "items_available": 1, "offset": 0},
                                                            {"items": [{"created_at": "",
                                                                        "head_uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzb",
                                                                        "link_class": "docker_image_repo+tag",
                                                                        "name": "arvados/jobs:"+arvados_cwl.__version__,
                                                                        "owner_uuid": "",
                                                                        "properties": {"image_timestamp": ""}}], "items_available": 1, "offset": 0},
                                                            {"items": [{"created_at": "",
                                                                        "head_uuid": "",
                                                                        "link_class": "docker_image_hash",
                                                                        "name": "123456",
                                                                        "owner_uuid": "",
                                                                        "properties": {"image_timestamp": ""}}], "items_available": 1, "offset": 0}
        )
        find_one_image_hash.return_value = "123456"

        arvrunner.api.collections().list().execute.side_effect = ({"items": [{"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzb",
                                                                              "owner_uuid": "",
                                                                              "manifest_text": "",
                                                                              "properties": ""
                                                                              }], "items_available": 1, "offset": 0},
                                                                  {"items": [{"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzb",
                                                                              "owner_uuid": "",
                                                                              "manifest_text": "",
                                                                              "properties": ""
                                                                          }], "items_available": 1, "offset": 0})
        arvrunner.api.collections().create().execute.return_value = {"uuid": ""}
        arvrunner.api.collections().get().execute.return_value = {"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzb",
                                                                  "portable_data_hash": "9999999999999999999999999999999b+99"}

        self.assertEqual("9999999999999999999999999999999b+99",
                         arvados_cwl.runner.arvados_jobs_image(arvrunner, "arvados/jobs:"+arvados_cwl.__version__, arvrunner.runtimeContext))


    @stubs()
    def test_submit_secrets(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
                "tests/wf/secret_wf.cwl", "tests/secret_test_job.yml"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        stubs.git_props["arv:gitPath"] = "sdk/cwl/tests/wf/secret_wf.cwl"
        stubs.git_info["http://arvados.org/cwl#gitPath"] = "sdk/cwl/tests/wf/secret_wf.cwl"

        expect_container = {
            "command": [
                "arvados-cwl-runner",
                "--local",
                "--api=containers",
                "--no-log-timestamps",
                "--disable-validate",
                "--disable-color",
                "--eval-timeout=20",
                '--thread-count=0',
                "--enable-reuse",
                "--collection-cache-size=256",
                '--output-name=Output from workflow secret_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                "--debug",
                "--on-error=continue",
                "/var/lib/cwl/workflow.json#main",
                "/var/lib/cwl/cwl.input.json"
            ],
            "container_image": "999999999999999999999999999999d3+99",
            "cwd": "/var/spool/cwl",
            "mounts": {
                "/var/lib/cwl/cwl.input.json": {
                    "content": {
                        "pw": {
                            "$include": "/secrets/s0"
                        }
                    },
                    "kind": "json"
                },
                "/var/lib/cwl/workflow.json": {
                    "content": {
                        "$graph": [
                            {
                                "arguments": [
                                    "md5sum",
                                    "example.conf"
                                ],
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
                                    {
                                        "id": "#secret_job.cwl/out",
                                        "type": "File",
                                        "outputBinding": {
                                              "glob": "hashed_example.txt"
                                        }
                                    }
                                ],
                                "stdout": "hashed_example.txt",
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
                                ]
                            },
                            {
                                "class": "Workflow",
                                "hints": [
                                    {
                                        "class": "DockerRequirement",
                                        "dockerPull": "debian:buster-slim",
                                        "http://arvados.org/cwl#dockerCollectionPDH": "999999999999999999999999999999d4+99"
                                    },
                                    {
                                        "class": "http://commonwl.org/cwltool#Secrets",
                                        "secrets": [
                                            "#main/pw"
                                        ]
                                    }
                                ],
                                "id": "#main",
                                "inputs": [
                                    {
                                        "id": "#main/pw",
                                        "type": "string"
                                    }
                                ],
                                "outputs": [
                                    {
                                        "id": "#main/out",
                                        "outputSource": "#main/step1/out",
                                        "type": "File"
                                    }
                                ],
                                "steps": [
                                    {
                                        "id": "#main/step1",
                                        "in": [
                                            {
                                                "id": "#main/step1/pw",
                                                "source": "#main/pw"
                                            }
                                        ],
                                        "out": [
                                            "#main/step1/out"
                                        ],
                                        "run": "#secret_job.cwl"
                                    }
                                ]
                            }
                        ],
                        "$namespaces": {
                            "cwltool": "http://commonwl.org/cwltool#"
                        },
                        "cwlVersion": "v1.0"
                    },
                    "kind": "json"
                },
                "/var/spool/cwl": {
                    "kind": "collection",
                    "writable": True
                },
                "stdout": {
                    "kind": "file",
                    "path": "/var/spool/cwl/cwl.output.json"
                }
            },
            "name": "secret_wf.cwl (%s)" % stubs.git_props["arv:gitDescribe"],
            "output_name": "Output from workflow secret_wf.cwl (%s)" % stubs.git_props["arv:gitDescribe"],
            "output_path": "/var/spool/cwl",
            "priority": 500,
            "properties": stubs.git_props,
            "runtime_constraints": {
                "API": True,
                "ram": 1342177280,
                "vcpus": 1
            },
            "secret_mounts": {
                "/secrets/s0": {
                    "content": "blorp",
                    "kind": "text"
                }
            },
            "state": "Committed",
            "use_existing": False
        }

        expect_container["mounts"]["/var/lib/cwl/workflow.json"]["content"].update(stubs.git_info)

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_request_uuid(self, stubs):
        stubs.api._rootDesc["remoteHosts"]["zzzzz"] = "123"
        stubs.expect_container_request_uuid = "zzzzz-xvhdp-yyyyyyyyyyyyyyy"

        stubs.api.container_requests().update().execute.return_value = {
            "uuid": stubs.expect_container_request_uuid,
            "container_uuid": "zzzzz-dz642-zzzzzzzzzzzzzzz",
            "state": "Queued"
        }

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--submit-request-uuid=zzzzz-xvhdp-yyyyyyyyyyyyyyy",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        stubs.api.container_requests().update.assert_called_with(
            uuid="zzzzz-xvhdp-yyyyyyyyyyyyyyy", body=JsonDiffMatcher(stubs.expect_container_spec))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_cluster_id(self, stubs):
        stubs.api._rootDesc["remoteHosts"]["zbbbb"] = "123"

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--submit-runner-cluster=zbbbb",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container), cluster_id="zbbbb")
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_validate_cluster_id(self, stubs):
        stubs.api._rootDesc["remoteHosts"]["zbbbb"] = "123"
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--submit-runner-cluster=zcccc",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)
        self.assertEqual(exited, 1)

    @stubs()
    def test_submit_validate_project_uuid(self, stubs):
        # Fails with bad cluster prefix
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--project-uuid=zzzzb-j7d0g-zzzzzzzzzzzzzzz",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)
        self.assertEqual(exited, 1)

        # Project lookup fails
        stubs.api.groups().get().execute.side_effect = Exception("Bad project")
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--project-uuid=zzzzz-j7d0g-zzzzzzzzzzzzzzx",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)
        self.assertEqual(exited, 1)

        # It should work this time because it is looking up a user (and only group is stubbed out to fail)
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--project-uuid=zzzzz-tpzed-zzzzzzzzzzzzzzx",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)
        self.assertEqual(exited, 0)


    @mock.patch("arvados.collection.CollectionReader")
    @stubs()
    def test_submit_uuid_inputs(self, stubs, collectionReader):
        collectionReader().exists.return_value = True
        collectionReader().find.return_value = arvados.arvfile.ArvadosFile(mock.MagicMock(), "file1.txt")
        def list_side_effect(**kwargs):
            m = mock.MagicMock()
            if "count" in kwargs:
                m.execute.return_value = {"items": [
                    {"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzz", "portable_data_hash": "99999999999999999999999999999998+99"}
                ]}
            else:
                m.execute.return_value = {"items": []}
            return m
        stubs.api.collections().list.side_effect = list_side_effect

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job_with_uuids.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container['mounts']['/var/lib/cwl/cwl.input.json']['content']['y']['basename'] = 'zzzzz-4zz18-zzzzzzzzzzzzzzz'
        expect_container['mounts']['/var/lib/cwl/cwl.input.json']['content']['y']['http://arvados.org/cwl#collectionUUID'] = 'zzzzz-4zz18-zzzzzzzzzzzzzzz'
        expect_container['mounts']['/var/lib/cwl/cwl.input.json']['content']['z']['listing'][0]['http://arvados.org/cwl#collectionUUID'] = 'zzzzz-4zz18-zzzzzzzzzzzzzzz'

        stubs.api.collections().list.assert_has_calls([
            mock.call(count='none',
                      filters=[['uuid', 'in', ['zzzzz-4zz18-zzzzzzzzzzzzzzz']]],
                      select=['uuid', 'portable_data_hash'])])
        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_mismatched_uuid_inputs(self, stubs):
        def list_side_effect(**kwargs):
            m = mock.MagicMock()
            if "count" in kwargs:
                m.execute.return_value = {"items": [
                    {"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzzz", "portable_data_hash": "99999999999999999999999999999997+99"}
                ]}
            else:
                m.execute.return_value = {"items": []}
            return m
        stubs.api.collections().list.side_effect = list_side_effect

        for infile in ("tests/submit_test_job_with_mismatched_uuids.json", "tests/submit_test_job_with_inconsistent_uuids.json"):
            capture_stderr = StringIO()
            cwltool_logger = logging.getLogger('cwltool')
            stderr_logger = logging.StreamHandler(capture_stderr)
            cwltool_logger.addHandler(stderr_logger)

            try:
                exited = arvados_cwl.main(
                    ["--submit", "--no-wait", "--api=containers", "--debug",
                        "tests/wf/submit_wf.cwl", infile],
                    stubs.capture_stdout, capture_stderr, api_client=stubs.api, keep_client=stubs.keep_client)

                self.assertEqual(exited, 1)
                self.assertRegex(
                    re.sub(r'[ \n]+', ' ', capture_stderr.getvalue()),
                    r"Expected collection uuid zzzzz-4zz18-zzzzzzzzzzzzzzz to be 99999999999999999999999999999998\+99 but API server reported 99999999999999999999999999999997\+99")
            finally:
                cwltool_logger.removeHandler(stderr_logger)

    @mock.patch("arvados.collection.CollectionReader")
    @stubs()
    def test_submit_unknown_uuid_inputs(self, stubs, collectionReader):
        collectionReader().find.return_value = arvados.arvfile.ArvadosFile(mock.MagicMock(), "file1.txt")
        capture_stderr = StringIO()

        cwltool_logger = logging.getLogger('cwltool')
        stderr_logger = logging.StreamHandler(capture_stderr)
        cwltool_logger.addHandler(stderr_logger)

        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job_with_uuids.json"],
            stubs.capture_stdout, capture_stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        try:
            self.assertEqual(exited, 1)
            self.assertRegex(
                capture_stderr.getvalue(),
                r"Collection\s*uuid\s*zzzzz-4zz18-zzzzzzzzzzzzzzz\s*not\s*found")
        finally:
            cwltool_logger.removeHandler(stderr_logger)

    @stubs(('submit_wf_process_properties.cwl', None))
    def test_submit_set_process_properties(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug",
                "tests/wf/submit_wf_process_properties.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)

        expect_container["mounts"]["/var/lib/cwl/workflow.json"]["content"]["$graph"][1]["hints"] = [
            {
                "class": "http://arvados.org/cwl#ProcessProperties",
                "processProperties": [
                    {"propertyName": "baz",
                     "propertyValue": "$(inputs.x.basename)"},
                    {"propertyName": "foo",
                     "propertyValue": "bar"},
                    {"propertyName": "quux",
                     "propertyValue": {
                         "q1": 1,
                         "q2": 2
                     }
                    }
                ],
            }
        ]
        expect_container["mounts"]["/var/lib/cwl/workflow.json"]["content"]["$namespaces"] = {
            "arv": "http://arvados.org/cwl#"
        }

        expect_container["properties"].update({
            "baz": "blorp.txt",
            "foo": "bar",
            "quux": {
                "q1": 1,
                "q2": 2
            }
        })

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)


    @stubs()
    def test_submit_enable_preemptible(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--enable-preemptible",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container['command'] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', '--on-error=continue',
                                       '--enable-preemptible',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_disable_preemptible(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--disable-preemptible",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container['command'] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', '--on-error=continue',
                                       '--disable-preemptible',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_prefer_cached_downloads(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--prefer-cached-downloads",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', "--on-error=continue", '--prefer-cached-downloads',
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_submit_container_varying_url_params(self, stubs):
        exited = arvados_cwl.main(
            ["--submit", "--no-wait", "--api=containers", "--debug", "--varying-url-params", "KeyId,Signature",
                "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["command"] = ['arvados-cwl-runner', '--local', '--api=containers',
                                       '--no-log-timestamps', '--disable-validate', '--disable-color',
                                       '--eval-timeout=20', '--thread-count=0',
                                       '--enable-reuse', "--collection-cache-size=256",
                                       '--output-name=Output from workflow submit_wf.cwl (%s)' % stubs.git_props["arv:gitDescribe"],
                                       '--debug', "--on-error=continue", "--varying-url-params=KeyId,Signature",
                                       '/var/lib/cwl/workflow.json#main', '/var/lib/cwl/cwl.input.json']

        stubs.api.container_requests().create.assert_called_with(
            body=JsonDiffMatcher(expect_container))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')
        self.assertEqual(exited, 0)


class TestCreateWorkflow(unittest.TestCase):
    existing_workflow_uuid = "zzzzz-7fd4e-validworkfloyml"
    expect_workflow = StripYAMLComments(
        open("tests/wf/expect_upload_wrapper.cwl").read().rstrip())
    expect_workflow_altname = StripYAMLComments(
        open("tests/wf/expect_upload_wrapper_altname.cwl").read().rstrip())

    def setUp(self):
        cwltool.process._names = set()
        arvados_cwl.arvdocker.arv_docker_clear_cache()

    def tearDown(self):
        root_logger = logging.getLogger('')

        # Remove existing RuntimeStatusLoggingHandlers if they exist
        handlers = [h for h in root_logger.handlers if not isinstance(h, arvados_cwl.executor.RuntimeStatusLoggingHandler)]
        root_logger.handlers = handlers

    @stubs()
    def test_create(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'
        stubs.api.groups().get().execute.return_value = {"group_class": "project"}

        exited = arvados_cwl.main(
            ["--create-workflow", "--debug",
             "--api=containers",
             "--project-uuid", project_uuid,
             "--disable-git",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        stubs.api.pipeline_templates().create.refute_called()
        stubs.api.container_requests().create.refute_called()

        body = {
            "workflow": {
                "owner_uuid": project_uuid,
                "name": "submit_wf.cwl",
                "description": "",
                "definition": self.expect_workflow,
            }
        }
        stubs.api.workflows().create.assert_called_with(
            body=JsonDiffMatcher(body))

        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_workflow_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_create_name(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'
        stubs.api.groups().get().execute.return_value = {"group_class": "project"}

        exited = arvados_cwl.main(
            ["--create-workflow", "--debug",
             "--api=containers",
             "--project-uuid", project_uuid,
             "--name", "testing 123",
             "--disable-git",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        stubs.api.pipeline_templates().create.refute_called()
        stubs.api.container_requests().create.refute_called()

        body = {
            "workflow": {
                "owner_uuid": project_uuid,
                "name": "testing 123",
                "description": "",
                "definition": self.expect_workflow_altname,
            }
        }
        stubs.api.workflows().create.assert_called_with(
            body=JsonDiffMatcher(body))

        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_workflow_uuid + '\n')
        self.assertEqual(exited, 0)


    @stubs()
    def test_update(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'
        stubs.api.workflows().get().execute.return_value = {"owner_uuid": project_uuid}

        exited = arvados_cwl.main(
            ["--update-workflow", self.existing_workflow_uuid,
             "--debug",
             "--disable-git",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        body = {
            "workflow": {
                "name": "submit_wf.cwl",
                "description": "",
                "definition": self.expect_workflow,
                "owner_uuid": project_uuid
            }
        }
        stubs.api.workflows().update.assert_called_with(
            uuid=self.existing_workflow_uuid,
            body=JsonDiffMatcher(body))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         self.existing_workflow_uuid + '\n')
        self.assertEqual(exited, 0)


    @stubs()
    def test_update_name(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'
        stubs.api.workflows().get().execute.return_value = {"owner_uuid": project_uuid}

        exited = arvados_cwl.main(
            ["--update-workflow", self.existing_workflow_uuid,
             "--debug", "--name", "testing 123",
             "--disable-git",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        body = {
            "workflow": {
                "name": "testing 123",
                "description": "",
                "definition": self.expect_workflow_altname,
                "owner_uuid": project_uuid
            }
        }
        stubs.api.workflows().update.assert_called_with(
            uuid=self.existing_workflow_uuid,
            body=JsonDiffMatcher(body))
        self.assertEqual(stubs.capture_stdout.getvalue(),
                         self.existing_workflow_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_create_collection_per_tool(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'
        stubs.api.groups().get().execute.return_value = {"group_class": "project"}

        exited = arvados_cwl.main(
            ["--create-workflow", "--debug",
             "--api=containers",
             "--project-uuid", project_uuid,
             "--disable-git",
             "tests/collection_per_tool/collection_per_tool.cwl"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        toolfile = "tests/collection_per_tool/collection_per_tool_wrapper.cwl"
        expect_workflow = StripYAMLComments(open(toolfile).read().rstrip())

        body = {
            "workflow": {
                "owner_uuid": project_uuid,
                "name": "collection_per_tool.cwl",
                "description": "",
                "definition": expect_workflow,
            }
        }
        stubs.api.workflows().create.assert_called_with(
            body=JsonDiffMatcher(body))

        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_workflow_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_create_with_imports(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'
        stubs.api.groups().get().execute.return_value = {"group_class": "project"}

        exited = arvados_cwl.main(
            ["--create-workflow", "--debug",
             "--api=containers",
             "--project-uuid", project_uuid,
             "tests/wf/feddemo/feddemo.cwl"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        stubs.api.pipeline_templates().create.refute_called()
        stubs.api.container_requests().create.refute_called()

        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_workflow_uuid + '\n')
        self.assertEqual(exited, 0)

    @stubs()
    def test_create_with_no_input(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'
        stubs.api.groups().get().execute.return_value = {"group_class": "project"}

        exited = arvados_cwl.main(
            ["--create-workflow", "--debug",
             "--api=containers",
             "--project-uuid", project_uuid,
             "tests/wf/revsort/revsort.cwl"],
            stubs.capture_stdout, sys.stderr, api_client=stubs.api)

        stubs.api.pipeline_templates().create.refute_called()
        stubs.api.container_requests().create.refute_called()

        self.assertEqual(stubs.capture_stdout.getvalue(),
                         stubs.expect_workflow_uuid + '\n')
        self.assertEqual(exited, 0)
