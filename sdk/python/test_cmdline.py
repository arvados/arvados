import os
import re
import subprocess
import unittest
import tempfile
import yaml

import arvados
import run_test_server

class ArvPutTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        try:
            del os.environ['KEEP_LOCAL_STORE']
        except KeyError:
            pass

        # Use the blob_signing_key from the Rails "test" configuration
        # to provision the Keep server.
        with open(os.path.join(os.path.dirname(__file__),
                               run_test_server.ARV_API_SERVER_DIR,
                               "config",
                               "application.yml")) as f:
            rails_config = yaml.load(f.read())
        config_blob_signing_key = rails_config["test"]["blob_signing_key"]
        run_test_server.run()
        run_test_server.run_keep(blob_signing_key=config_blob_signing_key,
                                 enforce_permissions=True)

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop()
        run_test_server.stop_keep()

    def test_ArvPutSignedManifest(self):
        run_test_server.authorize_with('active')
        for v in ["ARVADOS_API_HOST",
                  "ARVADOS_API_HOST_INSECURE",
                  "ARVADOS_API_TOKEN"]:
            os.environ[v] = arvados.config.settings()[v]

        datadir = tempfile.mkdtemp()
        with open(os.path.join(datadir, "foo"), "w") as f:
            f.write("The quick brown fox jumped over the lazy dog")
        p = subprocess.Popen(["arv-put", datadir],
                             stdout=subprocess.PIPE)
        (arvout, arverr) = p.communicate()
        self.assertEqual(arverr, None)

        # The manifest UUID returned by arv-put must be signed.
        manifest_uuid = arvout.strip()
        self.assertRegexpMatches(manifest_uuid, r'\+A[0-9a-f]+@[0-9a-f]{8}')

        # The manifest text stored in Keep must contain unsigned locators.
        m = arvados.Keep.get(manifest_uuid)
        self.assertEqual(m, ". 08a008a01d498c404b0c30852b39d3b8+44 0:44:foo\n")

        # The manifest text stored in the API server under the same
        # manifest UUID must use signed locators.
        api = arvados.api('v1', cache=False)
        c = api.collections().get(uuid=manifest_uuid).execute()
        self.assertRegexpMatches(
            c['manifest_text'],
            r'^\. 08a008a01d498c404b0c30852b39d3b8\+44\+A[0-9a-f]+@[0-9a-f]+ 0:44:foo\n')
