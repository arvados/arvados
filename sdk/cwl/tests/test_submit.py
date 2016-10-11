import arvados
import arvados.keep
import arvados.collection
import arvados_cwl
import copy
import cStringIO
import functools
import hashlib
import mock
import sys
import unittest
import json
import logging

from .matcher import JsonDiffMatcher


def stubs(func):
    @functools.wraps(func)
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    @mock.patch("arvados.collection.KeepClient")
    @mock.patch("arvados.events.subscribe")
    def wrapped(self, events, keep_client, keepdocker, *args, **kwargs):
        class Stubs:
            pass
        stubs = Stubs()
        stubs.events = events
        stubs.keepdocker = keepdocker
        stubs.keep_client = keep_client

        def putstub(p, **kwargs):
            return "%s+%i" % (hashlib.md5(p).hexdigest(), len(p))
        stubs.keep_client().put.side_effect = putstub
        stubs.keep_client.put.side_effect = putstub

        stubs.keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        stubs.fake_user_uuid = "zzzzz-tpzed-zzzzzzzzzzzzzzz"


        stubs.api = mock.MagicMock()
        stubs.api.users().current().execute.return_value = {
            "uuid": stubs.fake_user_uuid,
        }
        stubs.api.collections().list().execute.return_value = {"items": []}
        stubs.api.collections().create().execute.side_effect = ({
            "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz1",
            "portable_data_hash": "99999999999999999999999999999991+99",
            "manifest_text": ""
        }, {
            "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2",
            "portable_data_hash": "99999999999999999999999999999992+99",
            "manifest_text": "./tool 00000000000000000000000000000000+0 0:0:submit_tool.cwl 0:0:blub.txt"
        },
        {
            "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz4",
            "portable_data_hash": "99999999999999999999999999999994+99",
            "manifest_text": ""
        },
        {
            "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz5",
            "portable_data_hash": "99999999999999999999999999999995+99",
            "manifest_text": ""
        }        )
        stubs.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99", "manifest_text": "./tool 00000000000000000000000000000000+0 0:0:submit_tool.cwl 0:0:blub.txt"}

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
                'docker_image': 'arvados/jobs'
            },
            'script_parameters': {
                'x': {
                    'basename': 'blorp.txt',
                    'location': 'keep:99999999999999999999999999999994+99/blorp.txt',
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
                        "location": "keep:99999999999999999999999999999998+99/file1.txt"
                    }],
                    'class': 'Directory'
                },
                'cwl:tool':
                '99999999999999999999999999999991+99/wf/submit_wf.cwl'
            },
            'repository': 'arvados',
            'script_version': 'master',
            'script': 'cwl-runner'
        }

        stubs.expect_container_spec = {
            'priority': 1,
            'mounts': {
                '/var/spool/cwl': {
                    'writable': True,
                    'kind': 'collection'
                },
                '/var/lib/cwl/workflow': {
                    'portable_data_hash': '99999999999999999999999999999991+99',
                    'kind': 'collection'
                },
                'stdout': {
                    'path': '/var/spool/cwl/cwl.output.json',
                    'kind': 'file'
                },
                '/var/lib/cwl/job/cwl.input.json': {
                    'portable_data_hash': 'd20d7cddd1984f105dd3702c7f125afb+60/cwl.input.json',
                    'kind': 'collection'
                }
            },
            'state': 'Committed',
            'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
            'command': ['arvados-cwl-runner', '--local', '--api=containers', '/var/lib/cwl/workflow/submit_wf.cwl', '/var/lib/cwl/job/cwl.input.json'],
            'name': 'submit_wf.cwl',
            'container_image': '99999999999999999999999999999993+99',
            'output_path': '/var/spool/cwl',
            'cwd': '/var/spool/cwl',
            'runtime_constraints': {
                'API': True,
                'vcpus': 1,
                'ram': 268435456
            }
        }

        stubs.expect_workflow_uuid = "zzzzz-7fd4e-zzzzzzzzzzzzzzz"
        stubs.api.workflows().create().execute.return_value = {
            "uuid": stubs.expect_workflow_uuid,
        }

        return func(self, stubs, *args, **kwargs)
    return wrapped


class TestSubmit(unittest.TestCase):
    @stubs
    def test_submit(self, stubs):
        capture_stdout = cStringIO.StringIO()
        exited = arvados_cwl.main(
            ["--submit", "--no-wait",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            capture_stdout, sys.stderr, api_client=stubs.api)
        self.assertEqual(exited, 0)

        stubs.api.collections().create.assert_has_calls([
            mock.call(),
            mock.call(body={
                'manifest_text':
                './tool d51232d96b6116d964a69bfb7e0c73bf+450 '
                '0:16:blub.txt 16:434:submit_tool.cwl\n./wf '
                'cc2ffb940e60adf1b2b282c67587e43d+413 0:413:submit_wf.cwl\n',
                'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                'name': 'submit_wf.cwl',
            }, ensure_unique_name=True),
            mock.call().execute(),
            mock.call(body={'manifest_text': '. d41d8cd98f00b204e9800998ecf8427e+0 '
                            '0:0:blub.txt 0:0:submit_tool.cwl\n',
                            'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                            'replication_desired': None,
                            'name': 'New collection'
            }, ensure_unique_name=True),
            mock.call().execute(num_retries=4),
            mock.call(body={
                'manifest_text':
                '. 979af1245a12a1fed634d4222473bfdc+16 0:16:blorp.txt\n',
                'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                'name': '#',
            }, ensure_unique_name=True),
            mock.call().execute()])

        expect_job = copy.deepcopy(stubs.expect_job_spec)
        expect_job["owner_uuid"] = stubs.fake_user_uuid
        stubs.api.jobs().create.assert_called_with(
            body=expect_job,
            find_or_create=True)
        self.assertEqual(capture_stdout.getvalue(),
                         stubs.expect_job_uuid + '\n')

    @stubs
    def test_submit_with_project_uuid(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        exited = arvados_cwl.main(
            ["--submit", "--no-wait",
             "--project-uuid", project_uuid,
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            sys.stdout, sys.stderr, api_client=stubs.api)
        self.assertEqual(exited, 0)

        expect_body = copy.deepcopy(stubs.expect_job_spec)
        expect_body["owner_uuid"] = project_uuid
        stubs.api.jobs().create.assert_called_with(
            body=expect_body,
            find_or_create=True)

    @stubs
    def test_submit_container(self, stubs):
        capture_stdout = cStringIO.StringIO()
        try:
            exited = arvados_cwl.main(
                ["--submit", "--no-wait", "--api=containers", "--debug",
                 "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
                capture_stdout, sys.stderr, api_client=stubs.api, keep_client=stubs.keep_client)
            self.assertEqual(exited, 0)
        except:
            logging.exception("")

        stubs.api.collections().create.assert_has_calls([
            mock.call(),
            mock.call(body={
                'manifest_text':
                './tool d51232d96b6116d964a69bfb7e0c73bf+450 '
                '0:16:blub.txt 16:434:submit_tool.cwl\n./wf '
                'cc2ffb940e60adf1b2b282c67587e43d+413 0:413:submit_wf.cwl\n',
                'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                'name': 'submit_wf.cwl',
            }, ensure_unique_name=True),
            mock.call().execute(),
            mock.call(body={'manifest_text': '. d41d8cd98f00b204e9800998ecf8427e+0 '
                            '0:0:blub.txt 0:0:submit_tool.cwl\n',
                            'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                            'name': 'New collection',
                            'replication_desired': None,
            }, ensure_unique_name=True),
            mock.call().execute(num_retries=4),
            mock.call(body={
                'manifest_text':
                '. 979af1245a12a1fed634d4222473bfdc+16 0:16:blorp.txt\n',
                'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                'name': '#',
            }, ensure_unique_name=True),
            mock.call().execute()])

        expect_container = copy.deepcopy(stubs.expect_container_spec)
        expect_container["owner_uuid"] = stubs.fake_user_uuid
        stubs.api.container_requests().create.assert_called_with(
            body=expect_container)
        self.assertEqual(capture_stdout.getvalue(),
                         stubs.expect_container_request_uuid + '\n')


class TestCreateTemplate(unittest.TestCase):
    @stubs
    def test_create(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        capture_stdout = cStringIO.StringIO()

        exited = arvados_cwl.main(
            ["--create-template", "--debug",
             "--project-uuid", project_uuid,
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            capture_stdout, sys.stderr, api_client=stubs.api)
        self.assertEqual(exited, 0)

        stubs.api.pipeline_instances().create.refute_called()
        stubs.api.jobs().create.refute_called()

        expect_component = copy.deepcopy(stubs.expect_job_spec)
        expect_component['script_parameters']['x'] = {
            'dataclass': 'File',
            'required': True,
            'type': 'File',
            'value': '99999999999999999999999999999994+99/blorp.txt',
        }
        expect_component['script_parameters']['y'] = {
            'dataclass': 'Collection',
            'required': True,
            'type': 'Directory',
            'value': '99999999999999999999999999999998+99',
        }
        expect_component['script_parameters']['z'] = {
            'dataclass': 'Collection',
            'required': True,
            'type': 'Directory',
        }
        expect_template = {
            "components": {
                "submit_wf.cwl": expect_component,
            },
            "name": "submit_wf.cwl",
            "owner_uuid": project_uuid,
        }
        stubs.api.pipeline_templates().create.assert_called_with(
            body=JsonDiffMatcher(expect_template), ensure_unique_name=True)

        self.assertEqual(capture_stdout.getvalue(),
                         stubs.expect_pipeline_template_uuid + '\n')


class TestCreateWorkflow(unittest.TestCase):
    @stubs
    def test_create(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        capture_stdout = cStringIO.StringIO()

        exited = arvados_cwl.main(
            ["--create-workflow", "--debug",
             "--project-uuid", project_uuid,
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            capture_stdout, sys.stderr, api_client=stubs.api)
        self.assertEqual(exited, 0)

        stubs.api.pipeline_templates().create.refute_called()
        stubs.api.container_requests().create.refute_called()

        with open("tests/wf/expect_packed.cwl") as f:
            expect_workflow = f.read()

        body = {
            "workflow": {
                "owner_uuid": project_uuid,
                "name": "submit_wf.cwl",
                "description": "",
                "definition": expect_workflow
                }
        }
        stubs.api.workflows().create.assert_called_with(
            body=JsonDiffMatcher(body))

        self.assertEqual(capture_stdout.getvalue(),
                         stubs.expect_workflow_uuid + '\n')


class TestTemplateInputs(unittest.TestCase):
    expect_template = {
        "components": {
            "inputs_test.cwl": {
                'runtime_constraints': {
                    'docker_image': 'arvados/jobs',
                },
                'script_parameters': {
                    'cwl:tool':
                    '99999999999999999999999999999991+99/'
                    'wf/inputs_test.cwl',
                    'optionalFloatInput': None,
                    'fileInput': {
                        'type': 'File',
                        'dataclass': 'File',
                        'required': True,
                        'title': "It's a file; we expect to find some characters in it.",
                        'description': 'If there were anything further to say, it would be said here,\nor here.'
                    },
                    'floatInput': {
                        'type': 'float',
                        'dataclass': 'number',
                        'required': True,
                        'title': 'Floats like a duck',
                        'default': 0.1,
                        'value': 0.1,
                    },
                    'optionalFloatInput': {
                        'type': ['null', 'float'],
                        'dataclass': 'number',
                        'required': False,
                    },
                    'boolInput': {
                        'type': 'boolean',
                        'dataclass': 'boolean',
                        'required': True,
                        'title': 'True or false?',
                    },
                },
                'repository': 'arvados',
                'script_version': 'master',
                'script': 'cwl-runner',
            },
        },
        "name": "inputs_test.cwl",
    }

    @stubs
    def test_inputs_empty(self, stubs):
        exited = arvados_cwl.main(
            ["--create-template", "--no-wait",
             "tests/wf/inputs_test.cwl", "tests/order/empty_order.json"],
            cStringIO.StringIO(), sys.stderr, api_client=stubs.api)
        self.assertEqual(exited, 0)

        expect_template = copy.deepcopy(self.expect_template)
        expect_template["owner_uuid"] = stubs.fake_user_uuid

        stubs.api.pipeline_templates().create.assert_called_with(
            body=JsonDiffMatcher(expect_template), ensure_unique_name=True)

    @stubs
    def test_inputs(self, stubs):
        exited = arvados_cwl.main(
            ["--create-template", "--no-wait",
             "tests/wf/inputs_test.cwl", "tests/order/inputs_test_order.json"],
            cStringIO.StringIO(), sys.stderr, api_client=stubs.api)
        self.assertEqual(exited, 0)

        self.expect_template["owner_uuid"] = stubs.fake_user_uuid

        expect_template = copy.deepcopy(self.expect_template)
        expect_template["owner_uuid"] = stubs.fake_user_uuid
        params = expect_template[
            "components"]["inputs_test.cwl"]["script_parameters"]
        params["fileInput"]["value"] = '99999999999999999999999999999994+99/blorp.txt'
        params["floatInput"]["value"] = 1.234
        params["boolInput"]["value"] = True

        stubs.api.pipeline_templates().create.assert_called_with(
            body=JsonDiffMatcher(expect_template), ensure_unique_name=True)
