import arvados_cwl
import logging
import mock
import unittest
import os
import cwltool.process

if not os.getenv('ARVADOS_DEBUG'):
    logging.getLogger('arvados.cwl-runner').setLevel(logging.WARN)
    logging.getLogger('arvados.arv-run').setLevel(logging.WARN)


class TestJob(unittest.TestCase):

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    def test_run(self):
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("draft-3")

        tool = {
            "inputs": [],
            "outputs": [],
            "baseCommand": "ls"
        }
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, avsc_names=avsc_names, basedir="")
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), basedir=""):
            j.run()
            runner.api.jobs().create.assert_called_with(
                body={
                    'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                    'runtime_constraints': {},
                    'script_parameters': {
                        'tasks': [{
                            'task.env': {'TMPDIR': '$(task.tmpdir)'},
                            'command': ['ls']
                        }],
                    },
                    'script_version': 'master',
                    'minimum_script_version': '9e5b98e8f5f4727856b53447191f9c06e3da2ba6',
                    'repository': 'arvados',
                    'script': 'crunchrunner',
                    'runtime_constraints': {
                        'docker_image': 'arvados/jobs',
                        'min_cores_per_node': 1,
                        'min_ram_mb_per_node': 1024,
                        'min_scratch_mb_per_node': 2048 # tmpdirSize + outdirSize
                    }
                },
                find_or_create=True,
                filters=[['repository', '=', 'arvados'],
                         ['script', '=', 'crunchrunner'],
                         ['script_version', 'in git', '9e5b98e8f5f4727856b53447191f9c06e3da2ba6'],
                         ['docker_image_locator', 'in docker', 'arvados/jobs']]
            )

    # The test passes some fields in builder.resources
    # For the remaining fields, the defaults will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    def test_resource_requirements(self):
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("draft-3")

        tool = {
            "inputs": [],
            "outputs": [],
            "hints": [{
                "class": "ResourceRequirement",
                "coresMin": 3,
                "ramMin": 3000,
                "tmpdirMin": 4000
            }],
            "baseCommand": "ls"
        }
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, avsc_names=avsc_names)
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), basedir=""):
            j.run()
        runner.api.jobs().create.assert_called_with(
            body={
                'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                'runtime_constraints': {},
                'script_parameters': {
                    'tasks': [{
                        'task.env': {'TMPDIR': '$(task.tmpdir)'},
                        'command': ['ls']
                    }]
            },
            'script_version': 'master',
                'minimum_script_version': '9e5b98e8f5f4727856b53447191f9c06e3da2ba6',
                'repository': 'arvados',
                'script': 'crunchrunner',
                'runtime_constraints': {
                    'docker_image': 'arvados/jobs',
                    'min_cores_per_node': 3,
                    'min_ram_mb_per_node': 3000,
                    'min_scratch_mb_per_node': 5024 # tmpdirSize + outdirSize
                }
            },
            find_or_create=True,
            filters=[['repository', '=', 'arvados'],
                     ['script', '=', 'crunchrunner'],
                     ['script_version', 'in git', '9e5b98e8f5f4727856b53447191f9c06e3da2ba6'],
                     ['docker_image_locator', 'in docker', 'arvados/jobs']])

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

        arvjob = arvados_cwl.ArvadosJob(runner)
        arvjob.name = "testjob"
        arvjob.builder = mock.MagicMock()
        arvjob.output_callback = mock.MagicMock()
        arvjob.collect_outputs = mock.MagicMock()

        arvjob.done({
            "state": "Complete",
            "output": "99999999999999999999999999999993+99",
            "log": "99999999999999999999999999999994+99",
            "uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
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

        arvjob = arvados_cwl.ArvadosJob(runner)
        arvjob.name = "testjob"
        arvjob.builder = mock.MagicMock()
        arvjob.output_callback = mock.MagicMock()
        arvjob.collect_outputs = mock.MagicMock()

        arvjob.done({
            "state": "Complete",
            "output": "99999999999999999999999999999993+99",
            "log": "99999999999999999999999999999994+99",
            "uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        })

        api.collections().list.assert_has_calls([
            mock.call(),
            mock.call(filters=[['owner_uuid', '=', 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'],
                               ['portable_data_hash', '=', '99999999999999999999999999999993+99'],
                               ['name', '=', 'Output 9999999 of testjob']]),
            mock.call().execute(num_retries=0)])

        self.assertFalse(api.collections().create.called)
