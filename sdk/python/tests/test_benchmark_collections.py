import arvados
import sys

import run_test_server
import arvados_testutil as tutil
import manifest_examples
from performance.performance_profiler import profiled

class CollectionBenchmark(run_test_server.TestCaseWithServers,
                          tutil.ArvadosBaseTestCase,
                          manifest_examples.ManifestExamples):
    TEST_BLOCK_SIZE = 0

    @classmethod
    def list_recursive(cls, coll, parent_name=''):
        """Return a list of filenames in a [sub]collection.

        ["stream1/file1", "stream2/file1", ...]

        """

        items = []
        for name, item in coll.items():
            if callable(getattr(item, 'items', None)):
                # (ugh)
                items.extend(cls.list_recursive(item, parent_name+name+'/'))
            else:
                items.append(parent_name+name)
        return items

    @classmethod
    def setUpClass(cls):
        super(CollectionBenchmark, cls).setUpClass()
        run_test_server.authorize_with('active')
        cls.api_client = arvados.api('v1')
        cls.keep_client = arvados.KeepClient(api_client=cls.api_client,
                                             local_store=cls.local_store)

    @profiled
    def profile_new_collection_from_manifest(self, manifest_text):
        return arvados.collection.Collection(manifest_text)

    @profiled
    def profile_new_collection_from_server(self, uuid):
        return arvados.collection.Collection(uuid)

    @profiled
    def profile_new_collection_from_collection_files(self, src):
        dst = arvados.collection.Collection()
        with tutil.mock_keep_responses('x'*self.TEST_BLOCK_SIZE, 200):
            for name in self.list_recursive(src):
                with src.open(name) as srcfile:
                    with dst.open(name, 'w') as dstfile:
                        dstfile.write(srcfile.read())
            dst.save_new()

    @profiled
    def profile_collection_list_files(self, coll):
        return self.list_recursive(coll)

    def test_medium_sized_manifest(self):
        """Exercise manifest-handling code.

        Currently, this test puts undue emphasis on some code paths
        that don't reflect typical use because the contrived example
        manifest has some unusual characteristics:

        * Block size is zero.

        * Every block is identical, so block caching patterns are
          unrealistic.

        * Every file begins and ends at a block boundary.
        """
        specs = {
            'streams': 100,
            'files_per_stream': 100,
            'blocks_per_file': 20,
            'bytes_per_block': self.TEST_BLOCK_SIZE,
        }
        my_manifest = self.make_manifest(**specs)

        coll = self.profile_new_collection_from_manifest(my_manifest)

        coll.save_new()
        self.profile_new_collection_from_server(coll.manifest_locator())

        items = self.profile_collection_list_files(coll)
        self.assertEqual(len(items), specs['streams'] * specs['files_per_stream'])

        self.profile_new_collection_from_collection_files(coll)
