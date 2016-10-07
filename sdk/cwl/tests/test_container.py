import arvados_cwl
import logging
import mock
import unittest
import os
import functools
import cwltool.process
from schema_salad.ref_resolver import Loader

from schema_salad.ref_resolver import Loader

if not os.getenv('ARVADOS_DEBUG'):
    logging.getLogger('arvados.cwl-runner').setLevel(logging.WARN)
    logging.getLogger('arvados.arv-run').setLevel(logging.WARN)


class TestContainer(unittest.TestCase):

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_run(self, keepdocker):
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

        tool = {
            "inputs": [],
            "outputs": [],
            "baseCommand": "ls",
            "arguments": [{"valueFrom": "$(runtime.outdir)"}]
        }
        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess, api_client=runner.api)
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, work_api="containers", avsc_names=avsc_names,
                                                 basedir="", make_fs_access=make_fs_access, loader=Loader({}))
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), basedir="", name="test_run",
                             make_fs_access=make_fs_access, tmpdir="/tmp"):
            j.run()
            runner.api.container_requests().create.assert_called_with(
                body={
                    'environment': {
                        'HOME': '/var/spool/cwl',
                        'TMPDIR': '/tmp'
                    },
                    'name': 'test_run',
                    'runtime_constraints': {
                        'vcpus': 1,
                        'ram': 1073741824
                    }, 'priority': 1,
                    'mounts': {
                        '/var/spool/cwl': {'kind': 'tmp'}
                    },
                    'state': 'Committed',
                    'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                    'output_path': '/var/spool/cwl',
                    'container_image': '99999999999999999999999999999993+99',
                    'command': ['ls', '/var/spool/cwl'],
                    'cwd': '/var/spool/cwl'
                })

    # The test passes some fields in builder.resources
    # For the remaining fields, the defaults will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    def test_resource_requirements(self, keepdocker):
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

        keepdocker.return_value = [("zzzzz-4zz18-zzzzzzzzzzzzzz3", "")]
        runner.api.collections().get().execute.return_value = {
            "portable_data_hash": "99999999999999999999999999999993+99"}

        tool = {
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
        }
        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess, api_client=runner.api)
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, work_api="containers",
                                                 avsc_names=avsc_names, make_fs_access=make_fs_access,
                                                 loader=Loader({}))
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), basedir="", name="test_resource_requirements",
                             make_fs_access=make_fs_access, tmpdir="/tmp"):
            j.run()

        runner.api.container_requests().create.assert_called_with(
            body={
                'environment': {
                    'HOME': '/var/spool/cwl',
                    'TMPDIR': '/tmp'
                },
                'name': 'test_resource_requirements',
                'runtime_constraints': {
                    'vcpus': 3,
                    'ram': 3145728000,
                    'API': True,
                    'partition': ['blurb']
                }, 'priority': 1,
                'mounts': {
                    '/var/spool/cwl': {'kind': 'tmp'}
                },
                'state': 'Committed',
                'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                'output_path': '/var/spool/cwl',
                'container_image': '99999999999999999999999999999993+99',
                'command': ['ls'],
                'cwd': '/var/spool/cwl'
            })

    @mock.patch("arvados.collection.Collection")
    def test_done(self, col):
        api = mock.MagicMock()

        runner = mock.MagicMock()
        runner.api = api
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.num_retries = 0
        runner.ignore_docker_for_reuse = False

        col().open.return_value = []
        api.collections().list().execute.side_effect = ({"items": []},
                                                        {"items": [{"manifest_text": "XYZ"}]})

        arvjob = arvados_cwl.ArvadosContainer(runner)
        arvjob.name = "testjob"
        arvjob.builder = mock.MagicMock()
        arvjob.output_callback = mock.MagicMock()
        arvjob.collect_outputs = mock.MagicMock()
        arvjob.successCodes = [0]
        arvjob.outdir = "/var/spool/cwl"

        arvjob.done({
            "state": "Complete",
            "output": "99999999999999999999999999999993+99",
            "log": "99999999999999999999999999999994+99",
            "uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz",
            "exit_code": 0
        })

        api.collections().list.assert_has_calls([
            mock.call(),
            mock.call(filters=[['owner_uuid', '=', 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'],
                          ['portable_data_hash', '=', '99999999999999999999999999999993+99'],
                          ['name', '=', 'Output 9999999 of testjob']]),
            mock.call().execute(num_retries=0),
            mock.call(limit=1, filters=[['portable_data_hash', '=', '99999999999999999999999999999993+99']],
                 select=['manifest_text']),
            mock.call().execute(num_retries=0)])

        api.collections().create.assert_called_with(
            ensure_unique_name=True,
            body={'portable_data_hash': '99999999999999999999999999999993+99',
                  'manifest_text': 'XYZ',
                  'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                  'name': 'Output 9999999 of testjob'})

    @mock.patch("arvados.collection.Collection")
    def test_done_use_existing_collection(self, col):
        api = mock.MagicMock()

        runner = mock.MagicMock()
        runner.api = api
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.num_retries = 0

        col().open.return_value = []
        api.collections().list().execute.side_effect = ({"items": [{"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2"}]},)

        arvjob = arvados_cwl.ArvadosContainer(runner)
        arvjob.name = "testjob"
        arvjob.builder = mock.MagicMock()
        arvjob.output_callback = mock.MagicMock()
        arvjob.collect_outputs = mock.MagicMock()
        arvjob.successCodes = [0]
        arvjob.outdir = "/var/spool/cwl"

        arvjob.done({
            "state": "Complete",
            "output": "99999999999999999999999999999993+99",
            "log": "99999999999999999999999999999994+99",
            "uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz",
            "exit_code": 0
        })

        api.collections().list.assert_has_calls([
            mock.call(),
            mock.call(filters=[['owner_uuid', '=', 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'],
                               ['portable_data_hash', '=', '99999999999999999999999999999993+99'],
                               ['name', '=', 'Output 9999999 of testjob']]),
            mock.call().execute(num_retries=0)])

        self.assertFalse(api.collections().create.called)
