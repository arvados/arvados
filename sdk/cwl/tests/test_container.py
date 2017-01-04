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

            keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
            runner.api.collections().get().execute.return_value = {
                "portable_data_hash": "99999999999999999999999999999993+99"}

            document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

            tool = cmap({
                "inputs": [],
                "outputs": [],
                "baseCommand": "ls",
                "arguments": [{"valueFrom": "$(runtime.outdir)"}]
            })
            make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess, api_client=runner.api)
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
                            '/var/spool/cwl': {'kind': 'tmp'}
                        },
                        'state': 'Committed',
                        'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                        'output_path': '/var/spool/cwl',
                        'container_image': '99999999999999999999999999999993+99',
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
                "tmpdirMin": 4000
            }, {
                "class": "http://arvados.org/cwl#RuntimeConstraints",
                "keep_cache": 512
            }, {
                "class": "http://arvados.org/cwl#APIRequirement",
            }, {
                "class": "http://arvados.org/cwl#PartitionRequirement",
                "partition": "blurb"
            }],
            "baseCommand": "ls"
        })
        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess, api_client=runner.api)
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, work_api="containers",
                                                 avsc_names=avsc_names, make_fs_access=make_fs_access,
                                                 loader=Loader({}))
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), basedir="", name="test_resource_requirements",
                             make_fs_access=make_fs_access, tmpdir="/tmp"):
            j.run()

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
                    'keep_cache_ram': 512,
                    'API': True
                },
                'use_existing': True,
                'priority': 1,
                'mounts': {
                    '/var/spool/cwl': {'kind': 'tmp'}
                },
                'state': 'Committed',
                'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                'output_path': '/var/spool/cwl',
                'container_image': '99999999999999999999999999999993+99',
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

    @mock.patch("arvados.collection.Collection")
    def test_done(self, col):
        api = mock.MagicMock()

        runner = mock.MagicMock()
        runner.api = api
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.num_retries = 0
        runner.ignore_docker_for_reuse = False

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

        arvjob.collect_outputs.return_value = {"out": "stuff"}

        arvjob.done({
            "state": "Final",
            "log_uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz1",
            "output_uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2",
            "uuid": "zzzzz-xvhdp-zzzzzzzzzzzzzzz",
            "container_uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        })

        self.assertFalse(api.collections().create.called)

        arvjob.collect_outputs.assert_called_with("keep:abc+123")
        arvjob.output_callback.assert_called_with({"out": "stuff"}, "success")
