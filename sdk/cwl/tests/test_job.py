# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import functools
import json
import logging
import mock
import os
import unittest
import copy
import StringIO

import arvados
import arvados_cwl
import cwltool.process
from arvados.errors import ApiError
from schema_salad.ref_resolver import Loader
from schema_salad.sourceline import cmap
from .mock_discovery import get_rootDesc
from .matcher import JsonDiffMatcher, StripYAMLComments
from .test_container import CollectionMock

if not os.getenv('ARVADOS_DEBUG'):
    logging.getLogger('arvados.cwl-runner').setLevel(logging.WARN)
    logging.getLogger('arvados.arv-run').setLevel(logging.WARN)

class TestJob(unittest.TestCase):

    def helper(self, runner, enable_reuse=True):
        document_loader, avsc_names, schema_metadata, metaschema_loader = cwltool.process.get_schema("v1.0")

        make_fs_access=functools.partial(arvados_cwl.CollectionFsAccess,
                                         collection_cache=arvados_cwl.CollectionCache(runner.api, None, 0))
        loadingContext = arvados_cwl.context.ArvLoadingContext(
            {"avsc_names": avsc_names,
             "basedir": "",
             "make_fs_access": make_fs_access,
             "loader": Loader({}),
             "metadata": {"cwlVersion": "v1.0"},
             "makeTool": runner.arv_make_tool})
        runtimeContext = arvados_cwl.context.ArvRuntimeContext(
            {"work_api": "jobs",
             "basedir": "",
             "name": "test_run_job_"+str(enable_reuse),
             "make_fs_access": make_fs_access,
             "enable_reuse": enable_reuse,
             "priority": 500})

        return loadingContext, runtimeContext

    # The test passes no builder.resources
    # Hence the default resources will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch('arvados.commands.keepdocker.list_images_in_arv')
    def test_run(self, list_images_in_arv):
        for enable_reuse in (True, False):
            runner = mock.MagicMock()
            runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
            runner.ignore_docker_for_reuse = False
            runner.num_retries = 0

            list_images_in_arv.return_value = [["zzzzz-4zz18-zzzzzzzzzzzzzzz"]]
            runner.api.collections().get().execute.return_value = {"portable_data_hash": "99999999999999999999999999999993+99"}
            # Simulate reused job from another project so that we can check is a can_read
            # link is added.
            runner.api.jobs().create().execute.return_value = {
                'state': 'Complete' if enable_reuse else 'Queued',
                'owner_uuid': 'zzzzz-tpzed-yyyyyyyyyyyyyyy' if enable_reuse else 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                'uuid': 'zzzzz-819sb-yyyyyyyyyyyyyyy',
                'output': None,
            }

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
                runner.api.jobs().create.assert_called_with(
                    body=JsonDiffMatcher({
                        'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                        'runtime_constraints': {},
                        'script_parameters': {
                            'tasks': [{
                                'task.env': {'HOME': '$(task.outdir)', 'TMPDIR': '$(task.tmpdir)'},
                                'command': ['ls', '$(task.outdir)']
                            }],
                        },
                        'script_version': 'master',
                        'minimum_script_version': 'a3f2cb186e437bfce0031b024b2157b73ed2717d',
                        'repository': 'arvados',
                        'script': 'crunchrunner',
                        'runtime_constraints': {
                            'docker_image': 'arvados/jobs',
                            'min_cores_per_node': 1,
                            'min_ram_mb_per_node': 1024,
                            'min_scratch_mb_per_node': 2048 # tmpdirSize + outdirSize
                        }
                    }),
                    find_or_create=enable_reuse,
                    filters=[['repository', '=', 'arvados'],
                             ['script', '=', 'crunchrunner'],
                             ['script_version', 'in git', 'a3f2cb186e437bfce0031b024b2157b73ed2717d'],
                             ['docker_image_locator', 'in docker', 'arvados/jobs']]
                )
                if enable_reuse:
                    runner.api.links().create.assert_called_with(
                        body=JsonDiffMatcher({
                            'link_class': 'permission',
                            'name': 'can_read',
                            "tail_uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz",
                            "head_uuid": "zzzzz-819sb-yyyyyyyyyyyyyyy",
                        })
                    )
                    # Simulate an API excepction when trying to create a
                    # sharing link on the job
                    runner.api.links().create.side_effect = ApiError(
                        mock.MagicMock(return_value={'status': 403}),
                        'Permission denied')
                    j.run(runtimeContext)
                else:
                    assert not runner.api.links().create.called

    # The test passes some fields in builder.resources
    # For the remaining fields, the defaults will apply: {'cores': 1, 'ram': 1024, 'outdirSize': 1024, 'tmpdirSize': 1024}
    @mock.patch('arvados.commands.keepdocker.list_images_in_arv')
    def test_resource_requirements(self, list_images_in_arv):
        runner = mock.MagicMock()
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.num_retries = 0
        arvados_cwl.add_arv_hints()

        list_images_in_arv.return_value = [["zzzzz-4zz18-zzzzzzzzzzzzzzz"]]
        runner.api.collections().get().execute.return_vaulue = {"portable_data_hash": "99999999999999999999999999999993+99"}

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
                "keep_cache": 512,
                "outputDirType": "keep_output_dir"
            }, {
                "class": "http://arvados.org/cwl#APIRequirement",
            },
            {
                "class": "http://arvados.org/cwl#ReuseRequirement",
                "enableReuse": False
            }],
            "baseCommand": "ls",
            "id": "#",
            "class": "CommandLineTool"
        }

        loadingContext, runtimeContext = self.helper(runner)

        arvtool = arvados_cwl.ArvadosCommandTool(runner, tool, loadingContext)
        arvtool.formatgraph = None
        for j in arvtool.job({}, mock.MagicMock(), runtimeContext):
            j.run(runtimeContext)
        runner.api.jobs().create.assert_called_with(
            body=JsonDiffMatcher({
                'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                'runtime_constraints': {},
                'script_parameters': {
                    'tasks': [{
                        'task.env': {'HOME': '$(task.outdir)', 'TMPDIR': '$(task.tmpdir)'},
                        'task.keepTmpOutput': True,
                        'command': ['ls']
                    }]
            },
            'script_version': 'master',
                'minimum_script_version': 'a3f2cb186e437bfce0031b024b2157b73ed2717d',
                'repository': 'arvados',
                'script': 'crunchrunner',
                'runtime_constraints': {
                    'docker_image': 'arvados/jobs',
                    'min_cores_per_node': 3,
                    'min_ram_mb_per_node': 3512,     # ramMin + keep_cache
                    'min_scratch_mb_per_node': 5024, # tmpdirSize + outdirSize
                    'keep_cache_mb_per_task': 512
                }
            }),
            find_or_create=False,
            filters=[['repository', '=', 'arvados'],
                     ['script', '=', 'crunchrunner'],
                     ['script_version', 'in git', 'a3f2cb186e437bfce0031b024b2157b73ed2717d'],
                     ['docker_image_locator', 'in docker', 'arvados/jobs']])

    @mock.patch("arvados.collection.CollectionReader")
    def test_done(self, reader):
        api = mock.MagicMock()

        runner = mock.MagicMock()
        runner.api = api
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.num_retries = 0
        runner.ignore_docker_for_reuse = False

        reader().open.return_value = StringIO.StringIO(
            """2016-11-02_23:12:18 c97qk-8i9sb-cryqw2blvzy4yaj 13358 0 stderr 2016/11/02 23:12:18 crunchrunner: $(task.tmpdir)=/tmp/crunch-job-task-work/compute3.1/tmpdir
2016-11-02_23:12:18 c97qk-8i9sb-cryqw2blvzy4yaj 13358 0 stderr 2016/11/02 23:12:18 crunchrunner: $(task.outdir)=/tmp/crunch-job-task-work/compute3.1/outdir
2016-11-02_23:12:18 c97qk-8i9sb-cryqw2blvzy4yaj 13358 0 stderr 2016/11/02 23:12:18 crunchrunner: $(task.keep)=/keep
        """)
        api.collections().list().execute.side_effect = ({"items": []},
                                                        {"items": [{"manifest_text": "XYZ"}]},
                                                        {"items": []},
                                                        {"items": [{"manifest_text": "ABC"}]})

        arvjob = arvados_cwl.ArvadosJob(runner,
                                        mock.MagicMock(),
                                        {},
                                        None,
                                        [],
                                        [],
                                        "testjob")
        arvjob.output_callback = mock.MagicMock()
        arvjob.collect_outputs = mock.MagicMock()
        arvjob.collect_outputs.return_value = {"out": "stuff"}

        arvjob.done({
            "state": "Complete",
            "output": "99999999999999999999999999999993+99",
            "log": "99999999999999999999999999999994+99",
            "uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        })

        api.collections().list.assert_has_calls([
            mock.call(),
            # Output collection check
            mock.call(filters=[['owner_uuid', '=', 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'],
                          ['portable_data_hash', '=', '99999999999999999999999999999993+99'],
                          ['name', '=', 'Output 9999999 of testjob']]),
            mock.call().execute(num_retries=0),
            mock.call(limit=1, filters=[['portable_data_hash', '=', '99999999999999999999999999999993+99']],
                 select=['manifest_text']),
            mock.call().execute(num_retries=0),
            # Log collection's turn
            mock.call(filters=[['owner_uuid', '=', 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'],
                          ['portable_data_hash', '=', '99999999999999999999999999999994+99'],
                          ['name', '=', 'Log of zzzzz-8i9sb-zzzzzzzzzzzzzzz']]),
            mock.call().execute(num_retries=0),
            mock.call(limit=1, filters=[['portable_data_hash', '=', '99999999999999999999999999999994+99']],
                 select=['manifest_text']),
            mock.call().execute(num_retries=0)])

        api.collections().create.assert_has_calls([
            mock.call(ensure_unique_name=True,
                      body={'portable_data_hash': '99999999999999999999999999999993+99',
                            'manifest_text': 'XYZ',
                            'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                            'name': 'Output 9999999 of testjob'}),
            mock.call().execute(num_retries=0),
            mock.call(ensure_unique_name=True,
                      body={'portable_data_hash': '99999999999999999999999999999994+99',
                            'manifest_text': 'ABC',
                            'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz',
                            'name': 'Log of zzzzz-8i9sb-zzzzzzzzzzzzzzz'}),
            mock.call().execute(num_retries=0),
        ])

        arvjob.output_callback.assert_called_with({"out": "stuff"}, "success")

    @mock.patch("arvados.collection.CollectionReader")
    def test_done_use_existing_collection(self, reader):
        api = mock.MagicMock()

        runner = mock.MagicMock()
        runner.api = api
        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.num_retries = 0

        reader().open.return_value = StringIO.StringIO(
            """2016-11-02_23:12:18 c97qk-8i9sb-cryqw2blvzy4yaj 13358 0 stderr 2016/11/02 23:12:18 crunchrunner: $(task.tmpdir)=/tmp/crunch-job-task-work/compute3.1/tmpdir
2016-11-02_23:12:18 c97qk-8i9sb-cryqw2blvzy4yaj 13358 0 stderr 2016/11/02 23:12:18 crunchrunner: $(task.outdir)=/tmp/crunch-job-task-work/compute3.1/outdir
2016-11-02_23:12:18 c97qk-8i9sb-cryqw2blvzy4yaj 13358 0 stderr 2016/11/02 23:12:18 crunchrunner: $(task.keep)=/keep
        """)

        api.collections().list().execute.side_effect = (
            {"items": [{"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2"}]},
            {"items": [{"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2"}]},
        )

        arvjob = arvados_cwl.ArvadosJob(runner,
                                        mock.MagicMock(),
                                        {},
                                        None,
                                        [],
                                        [],
                                        "testjob")
        arvjob.output_callback = mock.MagicMock()
        arvjob.collect_outputs = mock.MagicMock()
        arvjob.collect_outputs.return_value = {"out": "stuff"}

        arvjob.done({
            "state": "Complete",
            "output": "99999999999999999999999999999993+99",
            "log": "99999999999999999999999999999994+99",
            "uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        })

        api.collections().list.assert_has_calls([
            mock.call(),
            # Output collection
            mock.call(filters=[['owner_uuid', '=', 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'],
                               ['portable_data_hash', '=', '99999999999999999999999999999993+99'],
                               ['name', '=', 'Output 9999999 of testjob']]),
            mock.call().execute(num_retries=0),
            # Log collection
            mock.call(filters=[['owner_uuid', '=', 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'],
                               ['portable_data_hash', '=', '99999999999999999999999999999994+99'],
                               ['name', '=', 'Log of zzzzz-8i9sb-zzzzzzzzzzzzzzz']]),
            mock.call().execute(num_retries=0)
        ])

        self.assertFalse(api.collections().create.called)

        arvjob.output_callback.assert_called_with({"out": "stuff"}, "success")


class TestWorkflow(unittest.TestCase):
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
             "metadata": {"cwlVersion": "v1.0"},
             "construct_tool_object": runner.arv_make_tool})
        runtimeContext = arvados_cwl.context.ArvRuntimeContext(
            {"work_api": "jobs",
             "basedir": "",
             "name": "test_run_wf_"+str(enable_reuse),
             "make_fs_access": make_fs_access,
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

        runner = arvados_cwl.ArvCwlRunner(api)
        self.assertEqual(runner.work_api, 'jobs')

        list_images_in_arv.return_value = [["zzzzz-4zz18-zzzzzzzzzzzzzzz"]]
        runner.api.collections().get().execute.return_vaulue = {"portable_data_hash": "99999999999999999999999999999993+99"}
        runner.api.collections().list().execute.return_vaulue = {"items": [{"portable_data_hash": "99999999999999999999999999999993+99"}]}

        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.num_retries = 0

        loadingContext, runtimeContext = self.helper(runner)

        tool, metadata = loadingContext.loader.resolve_ref("tests/wf/scatter2.cwl")
        metadata["cwlVersion"] = tool["cwlVersion"]

        mockc = mock.MagicMock()
        mockcollection.side_effect = lambda *args, **kwargs: CollectionMock(mockc, *args, **kwargs)
        mockcollectionreader().find.return_value = arvados.arvfile.ArvadosFile(mock.MagicMock(), "token.txt")

        arvtool = arvados_cwl.ArvadosWorkflow(runner, tool, loadingContext)
        arvtool.formatgraph = None
        it = arvtool.job({}, mock.MagicMock(), runtimeContext)

        it.next().run(runtimeContext)
        it.next().run(runtimeContext)

        with open("tests/wf/scatter2_subwf.cwl") as f:
            subwf = StripYAMLComments(f.read())

        runner.api.jobs().create.assert_called_with(
            body=JsonDiffMatcher({
                'minimum_script_version': 'a3f2cb186e437bfce0031b024b2157b73ed2717d',
                'repository': 'arvados',
                'script_version': 'master',
                'script': 'crunchrunner',
                'script_parameters': {
                    'tasks': [{'task.env': {
                        'HOME': '$(task.outdir)',
                        'TMPDIR': '$(task.tmpdir)'},
                               'task.vwd': {
                                   'workflow.cwl': '$(task.keep)/99999999999999999999999999999996+99/workflow.cwl',
                                   'cwl.input.yml': '$(task.keep)/99999999999999999999999999999996+99/cwl.input.yml'
                               },
                    'command': [u'cwltool', u'--no-container', u'--move-outputs', u'--preserve-entire-environment', u'workflow.cwl#main', u'cwl.input.yml'],
                    'task.stdout': 'cwl.output.json'}]},
                'runtime_constraints': {
                    'min_scratch_mb_per_node': 2048,
                    'min_cores_per_node': 1,
                    'docker_image': 'arvados/jobs',
                    'min_ram_mb_per_node': 1024
                },
                'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'}),
            filters=[['repository', '=', 'arvados'],
                     ['script', '=', 'crunchrunner'],
                     ['script_version', 'in git', 'a3f2cb186e437bfce0031b024b2157b73ed2717d'],
                     ['docker_image_locator', 'in docker', 'arvados/jobs']],
            find_or_create=True)

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

        runner = arvados_cwl.ArvCwlRunner(api)
        self.assertEqual(runner.work_api, 'jobs')

        list_images_in_arv.return_value = [["zzzzz-4zz18-zzzzzzzzzzzzzzz"]]
        runner.api.collections().get().execute.return_vaulue = {"portable_data_hash": "99999999999999999999999999999993+99"}
        runner.api.collections().list().execute.return_vaulue = {"items": [{"portable_data_hash": "99999999999999999999999999999993+99"}]}

        runner.project_uuid = "zzzzz-8i9sb-zzzzzzzzzzzzzzz"
        runner.ignore_docker_for_reuse = False
        runner.num_retries = 0

        loadingContext, runtimeContext = self.helper(runner)

        tool, metadata = loadingContext.loader.resolve_ref("tests/wf/echo-wf.cwl")
        metadata["cwlVersion"] = tool["cwlVersion"]

        mockcollection.side_effect = lambda *args, **kwargs: CollectionMock(mock.MagicMock(), *args, **kwargs)

        arvtool = arvados_cwl.ArvadosWorkflow(runner, tool, loadingContext)
        arvtool.formatgraph = None
        it = arvtool.job({}, mock.MagicMock(), runtimeContext)
        it.next().run(runtimeContext)
        it.next().run(runtimeContext)

        with open("tests/wf/echo-subwf.cwl") as f:
            subwf = StripYAMLComments(f.read())

        runner.api.jobs().create.assert_called_with(
            body=JsonDiffMatcher({
                'minimum_script_version': 'a3f2cb186e437bfce0031b024b2157b73ed2717d',
                'repository': 'arvados',
                'script_version': 'master',
                'script': 'crunchrunner',
                'script_parameters': {
                    'tasks': [{'task.env': {
                        'HOME': '$(task.outdir)',
                        'TMPDIR': '$(task.tmpdir)'},
                               'task.vwd': {
                                   'workflow.cwl': '$(task.keep)/99999999999999999999999999999996+99/workflow.cwl',
                                   'cwl.input.yml': '$(task.keep)/99999999999999999999999999999996+99/cwl.input.yml'
                               },
                    'command': [u'cwltool', u'--no-container', u'--move-outputs', u'--preserve-entire-environment', u'workflow.cwl#main', u'cwl.input.yml'],
                    'task.stdout': 'cwl.output.json'}]},
                'runtime_constraints': {
                    'min_scratch_mb_per_node': 4096,
                    'min_cores_per_node': 3,
                    'docker_image': 'arvados/jobs',
                    'min_ram_mb_per_node': 1024
                },
                'owner_uuid': 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'}),
            filters=[['repository', '=', 'arvados'],
                     ['script', '=', 'crunchrunner'],
                     ['script_version', 'in git', 'a3f2cb186e437bfce0031b024b2157b73ed2717d'],
                     ['docker_image_locator', 'in docker', 'arvados/jobs']],
            find_or_create=True)

    def test_default_work_api(self):
        arvados_cwl.add_arv_hints()

        api = mock.MagicMock()
        api._rootDesc = copy.deepcopy(get_rootDesc())
        del api._rootDesc.get('resources')['jobs']['methods']['create']
        runner = arvados_cwl.ArvCwlRunner(api)
        self.assertEqual(runner.work_api, 'containers')
