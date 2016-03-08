import unittest
import mock
import arvados_cwl

class TestJob(unittest.TestCase):
    def test_run(self):
        runner = mock.MagicMock()
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
            'runtime_constraints': {},
            'script_parameters': {
                'tasks': [{
                    'task.env': {'TMPDIR': '$(task.tmpdir)'},
                    'command': ['ls']
                }],
                'crunchrunner': '83db29f08544e1c319572a6bd971088a+140/crunchrunner'
            },
            'script_version':
            'master',
            'repository': 'arvados',
            'script': 'crunchrunner'
        }, find_or_create=True)


    def test_resource_requirements(self):
        runner = mock.MagicMock()
        tool = {
            "inputs": [],
            "outputs": [],
            "hints": [{
                "class": "ResourceRequirement",
                "minCores": 3,
                "ramMin": 3000,
                "tmpDirMin": 4000
            }],
            "baseCommand": "ls"
        }
        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool)
        arvtool.formatgraph = None
        for j in arvtool.job({}, "", mock.MagicMock()):
            j.run()
        runner.api.jobs().create.assert_called_with(body={
            'runtime_constraints': {},
            'script_parameters': {
                'tasks': [{
                    'task.env': {'TMPDIR': '$(task.tmpdir)'},
                    'command': ['ls']
                }],
                'crunchrunner': '83db29f08544e1c319572a6bd971088a+140/crunchrunner'
            },
            'script_version':
            'master',
            'repository': 'arvados',
            'script': 'crunchrunner',
            'runtime_constraints': {
                'min_cores_per_node': 3,
                'min_ram_mb_per_node': 3000,
                'min_scratch_mb_per_node': 4000
            }
        }, find_or_create=True)
