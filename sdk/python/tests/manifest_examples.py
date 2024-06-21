# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados

from . import arvados_testutil as tutil

class ManifestExamples(object):
    def make_manifest(self,
                      bytes_per_block=1,
                      blocks_per_file=1,
                      files_per_stream=1,
                      streams=1):
        datablip = b'x' * bytes_per_block
        data_loc = tutil.str_keep_locator(datablip)
        with tutil.mock_keep_responses(data_loc, 200):
            coll = arvados.collection.Collection()
            for si in range(0, streams):
                for fi in range(0, files_per_stream):
                    with coll.open("stream{}/file{}.txt".format(si, fi), 'wb') as f:
                        for bi in range(0, blocks_per_file):
                            f.write(datablip)
            return coll.manifest_text()
