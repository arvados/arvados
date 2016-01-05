from __future__ import print_function

import arvados
import collections
import functools
import gzip
import re
import sys


class Summarizer(object):
    def __init__(self, args):
        self.args = args

    def run(self):
        # stats_max: {category: {stat: val}}
        self.stats_max = collections.defaultdict(
            functools.partial(collections.defaultdict,
                              lambda: float('-Inf')))
        # task_stats: {task_id: {category: {stat: val}}}
        self.task_stats = collections.defaultdict(
            functools.partial(collections.defaultdict, dict))
        for line in self._logdata():
            m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) success in (?P<elapsed>\d+) seconds', line)
            if m:
                task_id = m.group('seq')
                elapsed = int(m.group('elapsed'))
                self.task_stats[task_id]['time'] = {'elapsed': elapsed}
                if elapsed > self.stats_max['time']['elapsed']:
                    self.stats_max['time']['elapsed'] = elapsed
                continue
            m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) stderr crunchstat: (?P<category>\S+) (?P<current>.*?)( -- interval (?P<interval>.*))?\n', line)
            if not m:
                continue
            if m.group('category').endswith(':'):
                # "notice:" etc.
                continue
            task_id = m.group('seq')
            this_interval_s = None
            for group in ['current', 'interval']:
                if not m.group(group):
                    continue
                category = m.group('category')
                words = m.group(group).split(' ')
                stats = {}
                for val, stat in zip(words[::2], words[1::2]):
                    if '.' in val:
                        stats[stat] = float(val)
                    else:
                        stats[stat] = int(val)
                if 'user' in stats or 'sys' in stats:
                    stats['user+sys'] = stats.get('user', 0) + stats.get('sys', 0)
                if 'tx' in stats or 'rx' in stats:
                    stats['tx+rx'] = stats.get('tx', 0) + stats.get('rx', 0)
                for stat, val in stats.iteritems():
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
                    else:
                        self.task_stats[task_id][category][stat] = val
                    if val > self.stats_max[category][stat]:
                        self.stats_max[category][stat] = val

    def report(self):
        return "\n".join(self._report_gen()) + "\n"

    def _report_gen(self):
        job_tot = collections.defaultdict(
            functools.partial(collections.defaultdict, int))
        for task_id, task_stat in self.task_stats.iteritems():
            for category, stat_last in task_stat.iteritems():
                for stat, val in stat_last.iteritems():
                    if stat in ['cpus', 'cache', 'swap', 'rss']:
                        # meaningless stats like 16 cpu cores x 5 tasks = 80
                        continue
                    job_tot[category][stat] += val
        yield "\t".join(['category', 'metric', 'task_max', 'task_max_rate', 'job_total'])
        for category, stat_max in sorted(self.stats_max.iteritems()):
            for stat, val in sorted(stat_max.iteritems()):
                if stat.endswith('__rate'):
                    continue
                max_rate = self._format(stat_max.get(stat+'__rate', '-'))
                val = self._format(val)
                tot = self._format(job_tot[category].get(stat, '-'))
                yield "\t".join([category, stat, str(val), max_rate, tot])
        for args in (
                ('Max CPU time spent by a single task: {}s',
                 self.stats_max['cpu']['user+sys'],
                 None),
                ('Max CPU usage in a single interval: {}%',
                 self.stats_max['cpu']['user+sys__rate'],
                 lambda x: x * 100),
                ('Overall CPU usage: {}%',
                 job_tot['cpu']['user+sys'] / job_tot['time']['elapsed'],
                 lambda x: x * 100),
                ('Max memory used by a single task: {}GB',
                 self.stats_max['mem']['rss'],
                 lambda x: x / 1e9),
                ('Max network traffic in a single task: {}GB',
                 self.stats_max['net:eth0']['tx+rx'],
                 lambda x: x / 1e9),
                ('Max network speed in a single interval: {}MB/s',
                 self.stats_max['net:eth0']['tx+rx__rate'],
                 lambda x: x / 1e6)):
            format_string, val, transform = args
            if val == float('-Inf'):
                continue
            if transform:
                val = transform(val)
            yield "# "+format_string.format(self._format(val))

    def _format(self, val):
        """Return a string representation of a stat.

        {:.2f} for floats, default format for everything else."""
        if isinstance(val, float):
            return '{:.2f}'.format(val)
        else:
            return '{}'.format(val)

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
