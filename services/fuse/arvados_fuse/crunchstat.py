# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import sys
import time

class Stat(object):
    def __init__(self, prefix, interval,
                 egr_name, ing_name,
                 egr_func, ing_func):
        self.prefix = prefix
        self.interval = interval
        self.egr_name = egr_name
        self.ing_name = ing_name
        self.egress = egr_func
        self.ingress = ing_func
        self.egr_prev = self.egress()
        self.ing_prev = self.ingress()

    def update(self):
        egr = self.egress()
        ing = self.ingress()

        delta = " -- interval %.4f seconds %d %s %d %s" % (self.interval,
                                                           egr - self.egr_prev,
                                                           self.egr_name,
                                                           ing - self.ing_prev,
                                                           self.ing_name)

        sys.stderr.write("crunchstat: %s %d %s %d %s%s\n" % (self.prefix,
                                                             egr,
                                                             self.egr_name,
                                                             ing,
                                                             self.ing_name,
                                                             delta))

        self.egr_prev = egr
        self.ing_prev = ing


def statlogger(interval, keep, ops):
    calls = Stat("keepcalls", interval, "put", "get",
                 keep.put_counter.get,
                 keep.get_counter.get)
    net = Stat("net:keep0", interval, "tx", "rx",
               keep.upload_counter.get,
               keep.download_counter.get)
    cache = Stat("keepcache", interval, "hit", "miss",
               keep.hits_counter.get,
               keep.misses_counter.get)
    fuseops = Stat("fuseops", interval,"write", "read",
                   ops.write_ops_counter.get,
                   ops.read_ops_counter.get)
    blk = Stat("blkio:0:0", interval, "write", "read",
               ops.write_counter.get,
               ops.read_counter.get)

    while True:
        time.sleep(interval)
        calls.update()
        net.update()
        cache.update()
        fuseops.update()
        blk.update()


