# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import arvados
import sys

from . import run_test_server
from . import arvados_testutil as tutil
from . import manifest_examples
from .performance.performance_profiler import profiled

class CollectionBenchmark(run_test_server.TestCaseWithServers,
                          tutil.ArvadosBaseTestCase,
                          manifest_examples.ManifestExamples):
    MAIN_SERVER = {}
    TEST_BLOCK_SIZE = 0

    @classmethod
    def list_recursive(cls, coll, parent_name=None):
        if parent_name is None:
            current_name = coll.stream_name()
        else:
            current_name = '{}/{}'.format(parent_name, coll.name)
        try:
            for name in coll:
                for item in cls.list_recursive(coll[name], current_name):
                    yield item
        except TypeError:
            yield current_name

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
    def profile_new_collection_copying_bytes_from_collection(self, src):
        dst = arvados.collection.Collection()
        with tutil.mock_keep_responses('x'*self.TEST_BLOCK_SIZE, 200):
            for name in self.list_recursive(src):
                with src.open(name, 'rb') as srcfile, dst.open(name, 'wb') as dstfile:
                    dstfile.write(srcfile.read())
            dst.save_new()

    @profiled
    def profile_new_collection_copying_files_from_collection(self, src):
        dst = arvados.collection.Collection()
        with tutil.mock_keep_responses('x'*self.TEST_BLOCK_SIZE, 200):
            for name in self.list_recursive(src):
                dst.copy(name, name, src)
            dst.save_new()

    @profiled
    def profile_collection_list_files(self, coll):
        return sum(1 for name in self.list_recursive(coll))

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

        num_items = self.profile_collection_list_files(coll)
        self.assertEqual(num_items, specs['streams'] * specs['files_per_stream'])

        self.profile_new_collection_copying_bytes_from_collection(coll)

        self.profile_new_collection_copying_files_from_collection(coll)
