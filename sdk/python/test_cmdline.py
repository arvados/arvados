import os
import subprocess
import unittest
import tempfile
import yaml

import apiclient
import arvados
import run_test_server

# ArvPutTest exercises arv-put behavior on the command line.
#
# Existing tests:
#
# ArvPutSignedManifest runs "arv-put foo" and then attempts to get
#   the newly created manifest from the API server, testing to confirm
#   that the block locators in the returned manifest are signed.
#
# TODO(twp): decide whether this belongs better in test_collections,
# since it chiefly exercises behavior in arvados.collection.CollectionWriter.
# Leaving it here for the time being because we may want to add more
# tests for arv-put command line behavior.

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

        # Before doing anything, demonstrate that the collection
        # we're about to create is not present in our test fixture.
        api = arvados.api('v1', cache=False)
        manifest_uuid = "00b4e9f40ac4dd432ef89749f1c01e74+47"
        with self.assertRaises(apiclient.errors.HttpError):
            notfound = api.collections().get(uuid=manifest_uuid).execute()
        
        datadir = tempfile.mkdtemp()
        with open(os.path.join(datadir, "foo"), "w") as f:
            f.write("The quick brown fox jumped over the lazy dog")
        p = subprocess.Popen(["./bin/arv-put", datadir],
                             stdout=subprocess.PIPE)
        (arvout, arverr) = p.communicate()
        self.assertEqual(p.returncode, 0)
        self.assertEqual(arverr, None)
        self.assertEqual(arvout.strip(), manifest_uuid)

        # The manifest text stored in the API server under the same
        # manifest UUID must use signed locators.
        c = api.collections().get(uuid=manifest_uuid).execute()
        self.assertRegexpMatches(
            c['manifest_text'],
            r'^\. 08a008a01d498c404b0c30852b39d3b8\+44\+A[0-9a-f]+@[0-9a-f]+ 0:44:foo\n')

        os.remove(os.path.join(datadir, "foo"))
        os.rmdir(datadir)
