# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import sys
import time
from collections import namedtuple

Stat = namedtuple("Stat", ['name', 'get'])

class StatWriter(object):    
    def __init__(self, prefix, interval, stats):
        self.prefix = prefix
        self.interval = interval
        self.stats = stats
        self.previous_stats = []
        self.update_previous_stats()

    def update_previous_stats(self):
        self.previous_stats = [stat.get() for stat in self.stats]

    def update(self):
        def append_by_type(string, name, value):
            if type(value) is float:
                string += " %.6f %s" % (value, name)
            else:
                string += " %s %s" % (str(value), name)
            return string

        out = "crunchstat: %s" % self.prefix
        delta = "-- interval %.4f seconds" % self.interval
        for i, stat in enumerate(self.stats):
            value = stat.get()
            diff = value - self.previous_stats[i]
            delta = append_by_type(delta, stat.name, diff)
            out = append_by_type(out, stat.name, value)

        sys.stderr.write("%s %s\n" % (out, delta))
        self.update_previous_stats()

def statlogger(interval, keep, ops):
    calls = StatWriter("keepcalls", interval, [
        Stat("put", keep.put_counter.get), 
        Stat("get", keep.get_counter.get)
    ])
    net = StatWriter("net:keep0", interval, [
        Stat("tx", keep.upload_counter.get),
        Stat("rx", keep.download_counter.get)
    ])
    cache = StatWriter("keepcache", interval, [
        Stat("hit", keep.hits_counter.get), 
        Stat("miss", keep.misses_counter.get)
    ])
    fuseops = StatWriter("fuseops", interval, [
        Stat("write", ops.write_ops_counter.get), 
        Stat("read", ops.read_ops_counter.get)
    ])
    fusetime = StatWriter("fuseopstime", interval, [
        Stat("seconds", ops.fuse_ops_total_time)
    ])
    blk = StatWriter("blkio:0:0", interval, [
        Stat("write", ops.write_counter.get),
        Stat("read", ops.read_counter.get)
    ])

    while True:
        time.sleep(interval)
        calls.update()
        net.update()
        cache.update()
        fuseops.update()
        fusetime.update()
        blk.update()


