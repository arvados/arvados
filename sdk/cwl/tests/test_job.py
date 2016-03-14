import unittest
import mock
import arvados_cwl

class TestJob(unittest.TestCase):

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    def test_run(self):
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        tool = {
            "inputs": [],
            "outputs": [],
            "baseCommand": "ls"
        }
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool)
        arvtool.formatgraph = None
        for j in arvtool.job({}, "", mock.MagicMock()):
            j.run()
        runner.api.jobs().create.assert_called_with(body={
            'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
            'runtime_constraints': {},
            'script_parameters': {
                'tasks': [{
                    'task.env': {'TMPDIR': '$(task.tmpdir)'},
                    'command': ['ls']
                }],
                'crunchrunner': '83db29f08544e1c319572a6bd971088a+140/crunchrunner'
            },
            'script_version': 'master',
            'minimum_script_version': '9e5b98e8f5f4727856b53447191f9c06e3da2ba6',
            'repository': 'arvados',
            'script': 'crunchrunner',
            'runtime_constraints': {
                'min_cores_per_node': 1,
                'min_ram_mb_per_node': 1024,
                'min_scratch_mb_per_node': 2048 # tmpdirSize + outdirSize
            }
        }, find_or_create=True)

    # The test passes some fields in builder.resources
    # For the remaining fields, the defaults will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    def test_resource_requirements(self):
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
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
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool)
        arvtool.formatgraph = None
        for j in arvtool.job({}, "", mock.MagicMock()):
            j.run()
        runner.api.jobs().create.assert_called_with(body={
            'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
            'runtime_constraints': {},
            'script_parameters': {
                'tasks': [{
                    'task.env': {'TMPDIR': '$(task.tmpdir)'},
                    'command': ['ls']
                }],
                'crunchrunner': '83db29f08544e1c319572a6bd971088a+140/crunchrunner'
            },
            'script_version': 'master',
            'minimum_script_version': '9e5b98e8f5f4727856b53447191f9c06e3da2ba6',
            'repository': 'arvados',
            'script': 'crunchrunner',
            'runtime_constraints': {
                'min_cores_per_node': 3,
                'min_ram_mb_per_node': 3000,
                'min_scratch_mb_per_node': 5024 # tmpdirSize + outdirSize
            }
        }, find_or_create=True)
