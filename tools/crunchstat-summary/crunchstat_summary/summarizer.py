from __future__ import print_function

import arvados
import gzip
import re
import sys


class Summarizer(object):
    def __init__(self, args):
        self.args = args

    def run(self):
        stats_max = {}
        for line in self._logdata():
            m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) stderr crunchstat: (?P<category>\S+) (?P<current>.*?)( -- interval (?P<interval>.*))?\n', line)
            if not m:
                continue
            if m.group('category').endswith(':'):
                # "notice:" etc.
                continue
            this_interval_s = None
            for group in ['current', 'interval']:
                if not m.group(group):
                    continue
                category = m.group('category')
                if category not in stats_max:
                    stats_max[category] = {}
                words = m.group(group).split(' ')
                for val, stat in zip(words[::2], words[1::2]):
                    if '.' in val:
                        val = float(val)
                    else:
                        val = int(val)
                    if group == 'interval':
                        if stat == 'seconds':
                            this_interval_s = val
                            continue
                        elif not (this_interval_s > 0):
                            print("BUG? interval stat given with duration {!r}".
                                  format(this_interval_s),
                                  file=sys.stderr)
                            continue
                        else:
                            stat = stat + '__rate'
                            val = val / this_interval_s
                    if val > stats_max[category].get(stat, float('-Inf')):
                        stats_max[category][stat] = val
        self.stats_max = stats_max

    def report(self):
        return "\n".join(self._report_gen()) + "\n"

    def _report_gen(self):
        yield "\t".join(['category', 'metric', 'max', 'max_rate'])
        for category, stat_max in self.stats_max.iteritems():
            for stat, val in stat_max.iteritems():
                if stat.endswith('__rate'):
                    continue
                if stat+'__rate' in stat_max:
                    max_rate = '{:.2f}'.format(stat_max[stat+'__rate'])
                else:
                    max_rate = '-'
                if isinstance(val, float):
                    val = '{:.2f}'.format(val)
                yield "\t".join([category, stat, str(val), max_rate])

    def _logdata(self):
        if self.args.log_file:
            if self.args.log_file.endswith('.gz'):
                return gzip.open(self.args.log_file)
            else:
                return open(self.args.log_file)
        elif self.args.job:
            arv = arvados.api('v1')
            job = arv.jobs().get(uuid=self.args.job).execute()
            if not job['log']:
                raise ValueError(
                    "job {} has no log; live summary not implemented".format(
                        self.args.job))
            collection = arvados.collection.CollectionReader(job['log'])
            filenames = [filename for filename in collection]
            if len(filenames) != 1:
                raise ValueError(
                    "collection {} has {} files; need exactly one".format(
                        job.log, len(filenames)))
            return collection.open(filenames[0])
        else:
            return sys.stdin
