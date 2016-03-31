import unittest
import mock
import arvados_cwl
import sys
import arvados
import arvados.keep
import arvados.collection
import hashlib

class TestSubmit(unittest.TestCase):
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    @mock.patch("arvados.collection.KeepClient")
    @mock.patch("arvados.events.subscribe")
    def test_submit(self, events, keep, keepdocker):
        api = mock.MagicMock()
        def putstub(p, **kwargs):
            return "%s+%i" % (hashlib.md5(p).hexdigest(), len(p))
        keep().put.side_effect = putstub
        keepdocker.return_value = True
        api.users().current().execute.return_value = {"uuid": "zzzzz-tpzed-zzzzzzzzzzzzzzz"}
        api.collections().list().execute.return_value = {"items": []}
        api.collections().create().execute.side_effect = ({"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz1",
                                                           "portable_data_hash": "99999999999999999999999999999991+99"},
                                                          {"uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2",
                                                           "portable_data_hash": "99999999999999999999999999999992+99"})
        api.jobs().create().execute.return_value = {"uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz", "state": "Queued"}

        arvados_cwl.main(["--debug", "--submit", "--no-wait", "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
                         sys.stdout, sys.stderr, api_client=api)

        api.collections().create.assert_has_calls([
            mock.call(),
            mock.call(body={'manifest_text': './tool 84ec4df683711de31b782505389a8843+429 0:16:blub.txt 16:413:submit_tool.cwl\n./wf 81d977a245a41b8e79859fbe00623fd0+344 0:344:submit_wf.cwl\n',
                            'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                            'name': 'submit_wf.cwl'
                        }, ensure_unique_name=True),
            mock.call().execute(),
            mock.call(body={'manifest_text': '. 979af1245a12a1fed634d4222473bfdc+16 0:16:blorp.txt\n',
                            'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                            'name': '#'
                        }, ensure_unique_name=True),
            mock.call().execute()])

        api.jobs().create.assert_called_with(
            body={
                'runtime_constraints': {
                    'docker_image': 'arvados/jobs'
                },
            'script_parameters': {
                'x': {
                    'path': '99999999999999999999999999999992+99/blorp.txt',
                    'class': 'File'
                },
                'cwl:tool': '99999999999999999999999999999991+99/wf/submit_wf.cwl'
            },
            'repository': 'arvados',
                'script_version': 'master',
                'script': 'cwl-runner'
            },
            find_or_create=True)
