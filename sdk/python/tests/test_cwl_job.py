import arvados
import unittest
import arvados_testutil as tutil
import arvados.commands.cwl_job as cwl_job
import run_test_server
import subprocess
import os
import stat

class CwlJobTestCase(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    KEEP_SERVER = {}
    ARV_GIT_SERVER = {}

    @classmethod
    def setUpClass(cls):
        super(CwlJobTestCase, cls).setUpClass()
        run_test_server.authorize_with("admin")
        cls.api_client = arvados.api('v1')
        os.chdir(os.path.join(run_test_server.ARVADOS_DIR, "docker"))
        subprocess.check_call(["./build.sh", "cwl-runner-image"])
        arvados.commands.keepdocker.upload_image(cls.api_client, ["arvados/cwl-runner"])

        repo = cls.api_client.repositories().create(body={"repository": {
            "owner_uuid": "zzzzz-tpzed-000000000000000",
            "name": "testrepo"
            }}).execute()

        os.mkdir(os.path.join(run_test_server.ARVADOS_DIR, "services/api/tmp/git/testrepo"))
        os.chdir(os.path.join(run_test_server.ARVADOS_DIR, "services/api/tmp/git/testrepo"))
        subprocess.check_call(["git", "init"])
        with open("foo", "w") as f:
            f.write("""#!/bin/sh
            echo foo
            """)
        os.chmod("foo", stat.S_IRUSR|stat.S_IRGRP|stat.S_IROTH|stat.S_IXUSR|stat.S_IXGRP|stat.S_IXOTH)
        subprocess.check_call(["git", "add", "foo"])
        subprocess.check_call(["git", "commit", "-mTest"])

    def test_parse_sinfo(self):
        nodes = cwl_job.parse_sinfo("""
16 compute0,compute2
8 compute1,compute[3-5],compute7
""")
        self.assertEqual({"compute0": {"slots": 16},
                          "compute2": {"slots": 16},
                          "compute1": {"slots": 8},
                          "compute3": {"slots": 8},
                          "compute4": {"slots": 8},
                          "compute5": {"slots": 8},
                          "compute7": {"slots": 8}},
        nodes)

    def test_make_slots(self):
        slots = cwl_job.make_slots({"compute0": {"slots": 2}, "compute1": {"slots": 4}})
        self.assertEqual({"compute0[0]": {"node": "compute0", "slot": 0, "task": None},
                          "compute0[1]": {"node": "compute0", "slot": 1, "task": None},
                          "compute1[0]": {"node": "compute1", "slot": 0, "task": None},
                          "compute1[1]": {"node": "compute1", "slot": 1, "task": None},
                          "compute1[2]": {"node": "compute1", "slot": 2, "task": None},
                          "compute1[3]": {"node": "compute1", "slot": 3, "task": None}},
                          slots)

    def test_run_job(self):
        job = CwlJobTestCase.api_client.jobs().create(body={"job": {
            "script": "foo",
            "script_version": "master",
            "script_parameters": { },
            "repository": "testrepo",
            "runtime_constraints": {
                "docker_image": "arvados/cwl-runner",
                "cwl_job": True
            } } }).execute()
        cwl_job.main(["--job", job["uuid"],
                      "--job-api-token", os.environ["ARVADOS_API_TOKEN"]])

        job2 = CwlJobTestCase.api_client.jobs().get(uuid=job["uuid"]).execute()
        self.assertEqual(job2["state"], "Complete")
