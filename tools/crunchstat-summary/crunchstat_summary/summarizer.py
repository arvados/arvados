from __future__ import print_function

import arvados
import collections
import crunchstat_summary.chartjs
import datetime
import functools
import itertools
import logging
import math
import re
import sys

logger = logging.getLogger(__name__)
logger.addHandler(logging.NullHandler())

# Recommend memory constraints that are this multiple of an integral
# number of GiB. (Actual nodes tend to be sold in sizes like 8 GiB
# that have amounts like 7.5 GiB according to the kernel.)
AVAILABLE_RAM_RATIO = 0.95


class Task(object):
    def __init__(self):
        self.starttime = None
        self.series = collections.defaultdict(list)


class Summarizer(object):
    existing_constraints = {}

    def __init__(self, logdata, label='job'):
        self._logdata = logdata
        self.label = label

    def run(self):
        # stats_max: {category: {stat: val}}
        self.stats_max = collections.defaultdict(
            functools.partial(collections.defaultdict,
                              lambda: float('-Inf')))
        # task_stats: {task_id: {category: {stat: val}}}
        self.task_stats = collections.defaultdict(
            functools.partial(collections.defaultdict, dict))
        self.tasks = collections.defaultdict(Task)
        for line in self._logdata:
            m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) success in (?P<elapsed>\d+) seconds', line)
            if m:
                task_id = m.group('seq')
                elapsed = int(m.group('elapsed'))
                self.task_stats[task_id]['time'] = {'elapsed': elapsed}
                if elapsed > self.stats_max['time']['elapsed']:
                    self.stats_max['time']['elapsed'] = elapsed
                continue
            m = re.search(r'^(?P<timestamp>\S+) \S+ \d+ (?P<seq>\d+) stderr crunchstat: (?P<category>\S+) (?P<current>.*?)( -- interval (?P<interval>.*))?\n', line)
            if not m:
                continue
            if m.group('category').endswith(':'):
                # "notice:" etc.
                continue
            elif m.group('category') == 'error':
                continue
            task_id = m.group('seq')
            timestamp = datetime.datetime.strptime(
                m.group('timestamp'), '%Y-%m-%d_%H:%M:%S')
            task = self.tasks[task_id]
            if not task.starttime:
                task.starttime = timestamp
            this_interval_s = None
            for group in ['current', 'interval']:
                if not m.group(group):
                    continue
                category = m.group('category')
                words = m.group(group).split(' ')
                stats = {}
                for val, stat in zip(words[::2], words[1::2]):
                    try:
                        if '.' in val:
                            stats[stat] = float(val)
                        else:
                            stats[stat] = int(val)
                    except ValueError as e:
                        raise ValueError(
                            'Error parsing {} stat in "{}": {!r}'.format(
                                stat, line, e))
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
                            logger.error(
                                "BUG? interval stat given with duration {!r}".
                                format(this_interval_s))
                            continue
                        else:
                            stat = stat + '__rate'
                            val = val / this_interval_s
                            if stat in ['user+sys__rate', 'tx+rx__rate']:
                                task.series[category, stat].append(
                                    (timestamp - task.starttime, val))
                    else:
                        if stat in ['rss']:
                            task.series[category, stat].append(
                                (timestamp - task.starttime, val))
                        self.task_stats[task_id][category][stat] = val
                    if val > self.stats_max[category][stat]:
                        self.stats_max[category][stat] = val
        self.job_tot = collections.defaultdict(
            functools.partial(collections.defaultdict, int))
        for task_id, task_stat in self.task_stats.iteritems():
            for category, stat_last in task_stat.iteritems():
                for stat, val in stat_last.iteritems():
                    if stat in ['cpus', 'cache', 'swap', 'rss']:
                        # meaningless stats like 16 cpu cores x 5 tasks = 80
                        continue
                    self.job_tot[category][stat] += val

    def text_report(self):
        return "\n".join(itertools.chain(
            self._text_report_gen(),
            self._recommend_gen())) + "\n"

    def html_report(self):
        return crunchstat_summary.chartjs.ChartJS(self.label, self.tasks).html()

    def _text_report_gen(self):
        yield "\t".join(['category', 'metric', 'task_max', 'task_max_rate', 'job_total'])
        for category, stat_max in sorted(self.stats_max.iteritems()):
            for stat, val in sorted(stat_max.iteritems()):
                if stat.endswith('__rate'):
                    continue
                max_rate = self._format(stat_max.get(stat+'__rate', '-'))
                val = self._format(val)
                tot = self._format(self.job_tot[category].get(stat, '-'))
                yield "\t".join([category, stat, str(val), max_rate, tot])
        for args in (
                ('Max CPU time spent by a single task: {}s',
                 self.stats_max['cpu']['user+sys'],
                 None),
                ('Max CPU usage in a single interval: {}%',
                 self.stats_max['cpu']['user+sys__rate'],
                 lambda x: x * 100),
                ('Overall CPU usage: {}%',
                 self.job_tot['cpu']['user+sys'] /
                 self.job_tot['time']['elapsed'],
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

    def _recommend_gen(self):
        return itertools.chain(
            self._recommend_cpu(),
            self._recommend_ram())

    def _recommend_cpu(self):
        """Recommend asking for 4 cores if max CPU usage was 333%"""

        cpu_max_rate = self.stats_max['cpu']['user+sys__rate']
        if cpu_max_rate == float('-Inf'):
            logger.warning('%s: no CPU usage data', self.label)
            return
        used_cores = int(math.ceil(cpu_max_rate))
        asked_cores =  self.existing_constraints.get('min_cores_per_node')
        if asked_cores is None or used_cores < asked_cores:
            yield (
                '#!! {} max CPU usage was {}% -- '
                'try runtime_constraints "min_cores_per_node":{}'
            ).format(
                self.label,
                int(math.ceil(cpu_max_rate*100)),
                int(used_cores))

    def _recommend_ram(self):
        """Recommend asking for (2048*0.95) MiB RAM if max rss was 1248 MiB"""

        used_ram = self.stats_max['mem']['rss']
        if used_ram == float('-Inf'):
            logger.warning('%s: no memory usage data', self.label)
            return
        used_ram = math.ceil(float(used_ram) / (1<<20))
        asked_ram = self.existing_constraints.get('min_ram_mb_per_node')
        if asked_ram is None or (
                math.ceil((used_ram/AVAILABLE_RAM_RATIO)/(1<<10)) <
                (asked_ram/AVAILABLE_RAM_RATIO)/(1<<10)):
            yield (
                '#!! {} max RSS was {} MiB -- '
                'try runtime_constraints "min_ram_mb_per_node":{}'
            ).format(
                self.label,
                int(used_ram),
                int(math.ceil((used_ram/AVAILABLE_RAM_RATIO)/(1<<10))*(1<<10)*AVAILABLE_RAM_RATIO))

    def _format(self, val):
        """Return a string representation of a stat.

        {:.2f} for floats, default format for everything else."""
        if isinstance(val, float):
            return '{:.2f}'.format(val)
        else:
            return '{}'.format(val)


class CollectionSummarizer(Summarizer):
    def __init__(self, collection_id):
        collection = arvados.collection.CollectionReader(collection_id)
        filenames = [filename for filename in collection]
        if len(filenames) != 1:
            raise ValueError(
                "collection {} has {} files; need exactly one".format(
                    collection_id, len(filenames)))
        super(CollectionSummarizer, self).__init__(
            collection.open(filenames[0]))


class JobSummarizer(CollectionSummarizer):
    def __init__(self, job):
        arv = arvados.api('v1')
        if isinstance(job, str):
            self.job = arv.jobs().get(uuid=job).execute()
        else:
            self.job = job
        self.label = self.job['uuid']
        self.existing_constraints = self.job.get('runtime_constraints', {})
        if not self.job['log']:
            raise ValueError(
                "job {} has no log; live summary not implemented".format(
                    self.job['uuid']))
        super(JobSummarizer, self).__init__(self.job['log'])


class PipelineSummarizer():
    def __init__(self, pipeline_instance_uuid):
        arv = arvados.api('v1')
        instance = arv.pipeline_instances().get(
            uuid=pipeline_instance_uuid).execute()
        self.summarizers = collections.OrderedDict()
        for cname, component in instance['components'].iteritems():
            if 'job' not in component:
                logger.warning(
                    "%s: skipping component with no job assigned", cname)
            elif component['job'].get('log') is None:
                logger.warning(
                    "%s: skipping job %s with no log available",
                    cname, component['job'].get('uuid'))
            else:
                logger.debug(
                    "%s: reading log from %s", cname, component['job']['log'])
                summarizer = JobSummarizer(component['job'])
                summarizer.label = cname
                self.summarizers[cname] = summarizer

    def run(self):
        for summarizer in self.summarizers.itervalues():
            summarizer.run()

    def text_report(self):
        txt = ''
        for cname, summarizer in self.summarizers.iteritems():
            txt += '### Summary for {} ({})\n'.format(
                cname, summarizer.job['uuid'])
            txt += summarizer.text_report()
            txt += '\n'
        return txt
