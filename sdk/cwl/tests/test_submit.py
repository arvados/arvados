import arvados
import arvados.keep
import arvados.collection
import arvados_cwl
import functools
import hashlib
import mock
import sys
import unittest

def stubs(func):
    @functools.wraps(func)
    @mock.patch("arvados.commands.keepdocker.list_images_in_arv")
    @mock.patch("arvados.collection.KeepClient")
    @mock.patch("arvados.events.subscribe")
    def wrapped(self, events, KeepClient, keepdocker, *args, **kwargs):
        class Stubs:
            pass
        stubs = Stubs()
        stubs.events = events
        stubs.KeepClient = KeepClient
        stubs.keepdocker = keepdocker

        def putstub(p, **kwargs):
            return "%s+%i" % (hashlib.md5(p).hexdigest(), len(p))
        stubs.KeepClient().put.side_effect = putstub

        stubs.keepdocker.return_value = True
        stubs.fake_user_uuid = "zzzzz-tpzed-zzzzzzzzzzzzzzz"

        stubs.api = mock.MagicMock()
        stubs.api.users().current().execute.return_value = {"uuid": stubs.fake_user_uuid}
        stubs.api.collections().list().execute.return_value = {"items": []}
        stubs.api.collections().create().execute.side_effect = ({
            "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz1",
            "portable_data_hash": "99999999999999999999999999999991+99",
        }, {
            "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz2",
            "portable_data_hash": "99999999999999999999999999999992+99",
        })
        stubs.api.jobs().create().execute.return_value = {
            "uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz",
            "state": "Queued",
        }
        stubs.expect_job_spec = {
            'owner_uuid': stubs.fake_user_uuid,
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
        }
        return func(self, stubs, *args, **kwargs)
    return wrapped


class TestSubmit(unittest.TestCase):
    @stubs
    def test_submit(self, stubs):
        arvados_cwl.main(
            ["--debug", "--submit", "--no-wait",
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            sys.stdout, sys.stderr, api_client=stubs.api)

        stubs.api.collections().create.assert_has_calls([
            mock.call(),
            mock.call(body={
                'manifest_text': './tool 84ec4df683711de31b782505389a8843+429 0:16:blub.txt 16:413:submit_tool.cwl\n./wf 81d977a245a41b8e79859fbe00623fd0+344 0:344:submit_wf.cwl\n',
                'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                'name': 'submit_wf.cwl',
            }, ensure_unique_name=True),
            mock.call().execute(),
            mock.call(body={
                'manifest_text': '. 979af1245a12a1fed634d4222473bfdc+16 0:16:blorp.txt\n',
                'owner_uuid': 'zzzzz-tpzed-zzzzzzzzzzzzzzz',
                'name': '#',
            }, ensure_unique_name=True),
            mock.call().execute()])

        stubs.api.jobs().create.assert_called_with(
            body=stubs.expect_job_spec,
            find_or_create=True)

    @stubs
    def test_submit_with_project_uuid(self, stubs):
        project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        arvados_cwl.main(
            ["--debug", "--submit", "--no-wait",
             "--project-uuid", project_uuid,
             "tests/wf/submit_wf.cwl", "tests/submit_test_job.json"],
            sys.stdout, sys.stderr, api_client=stubs.api)

        expect_body = stubs.expect_job_spec.copy()
        expect_body["owner_uuid"] = project_uuid
        stubs.api.jobs().create.assert_called_with(
            body=expect_body,
            find_or_create=True)
