from __future__ import print_function

import arvados
import collections
import crunchstat_summary.chartjs
import datetime
import functools
import itertools
import math
import re
import sys

from arvados.api import OrderedJsonModel
from crunchstat_summary import logger

# Recommend memory constraints that are this multiple of an integral
# number of GiB. (Actual nodes tend to be sold in sizes like 8 GiB
# that have amounts like 7.5 GiB according to the kernel.)
AVAILABLE_RAM_RATIO = 0.95


class Task(object):
    def __init__(self):
        self.starttime = None
        self.series = collections.defaultdict(list)


class Summarizer(object):
    def __init__(self, logdata, label=None, skip_child_jobs=False):
        self._logdata = logdata

        self.label = label
        self.starttime = None
        self.finishtime = None
        self._skip_child_jobs = skip_child_jobs

        # stats_max: {category: {stat: val}}
        self.stats_max = collections.defaultdict(
            functools.partial(collections.defaultdict,
                              lambda: float('-Inf')))
        # task_stats: {task_id: {category: {stat: val}}}
        self.task_stats = collections.defaultdict(
            functools.partial(collections.defaultdict, dict))

        self.seq_to_uuid = {}
        self.tasks = collections.defaultdict(Task)

        # We won't bother recommending new runtime constraints if the
        # constraints given when running the job are known to us and
        # are already suitable.  If applicable, the subclass
        # constructor will overwrite this with something useful.
        self.existing_constraints = {}

        logger.debug("%s: logdata %s", self.label, repr(logdata))

    def run(self):
        logger.debug("%s: parsing log data", self.label)
        for line in self._logdata:
            m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) job_task (?P<task_uuid>\S+)$', line)
            if m:
                seq = int(m.group('seq'))
                uuid = m.group('task_uuid')
                self.seq_to_uuid[seq] = uuid
                logger.debug('%s: seq %d is task %s', self.label, seq, uuid)
                continue

            m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) success in (?P<elapsed>\d+) seconds', line)
            if m:
                task_id = self.seq_to_uuid[int(m.group('seq'))]
                elapsed = int(m.group('elapsed'))
                self.task_stats[task_id]['time'] = {'elapsed': elapsed}
                if elapsed > self.stats_max['time']['elapsed']:
                    self.stats_max['time']['elapsed'] = elapsed
                continue

            m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) stderr Queued job (?P<uuid>\S+)$', line)
            if m:
                uuid = m.group('uuid')
                if self._skip_child_jobs:
                    logger.warning('%s: omitting stats from child job %s'
                                   ' because --skip-child-jobs flag is on',
                                   self.label, uuid)
                    continue
                logger.debug('%s: follow %s', self.label, uuid)
                child_summarizer = JobSummarizer(uuid)
                child_summarizer.stats_max = self.stats_max
                child_summarizer.task_stats = self.task_stats
                child_summarizer.tasks = self.tasks
                child_summarizer.run()
                logger.debug('%s: done %s', self.label, uuid)
                continue

            m = re.search(r'^(?P<timestamp>\S+) (?P<job_uuid>\S+) \d+ (?P<seq>\d+) stderr crunchstat: (?P<category>\S+) (?P<current>.*?)( -- interval (?P<interval>.*))?\n', line)
            if not m:
                continue

            if self.label is None:
                self.label = m.group('job_uuid')
                logger.debug('%s: using job uuid as label', self.label)
            if m.group('category').endswith(':'):
                # "stderr crunchstat: notice: ..."
                continue
            elif m.group('category') in ('error', 'caught'):
                continue
            elif m.group('category') == 'read':
                # "stderr crunchstat: read /proc/1234/net/dev: ..."
                # (crunchstat formatting fixed, but old logs still say this)
                continue
            task_id = self.seq_to_uuid[int(m.group('seq'))]
            task = self.tasks[task_id]

            # Use the first and last crunchstat timestamps as
            # approximations of starttime and finishtime.
            timestamp = datetime.datetime.strptime(
                m.group('timestamp'), '%Y-%m-%d_%H:%M:%S')
            if not task.starttime:
                task.starttime = timestamp
                logger.debug('%s: task %s starttime %s',
                             self.label, task_id, timestamp)
            task.finishtime = timestamp

            if not self.starttime:
                self.starttime = timestamp
            self.finishtime = timestamp

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
        logger.debug('%s: done parsing', self.label)

        self.job_tot = collections.defaultdict(
            functools.partial(collections.defaultdict, int))
        for task_id, task_stat in self.task_stats.iteritems():
            for category, stat_last in task_stat.iteritems():
                for stat, val in stat_last.iteritems():
                    if stat in ['cpus', 'cache', 'swap', 'rss']:
                        # meaningless stats like 16 cpu cores x 5 tasks = 80
                        continue
                    self.job_tot[category][stat] += val
        logger.debug('%s: done totals', self.label)

    def long_label(self):
        label = self.label
        if self.finishtime:
            label += ' -- elapsed time '
            s = (self.finishtime - self.starttime).total_seconds()
            if s > 86400:
                label += '{}d'.format(int(s/86400))
            if s > 3600:
                label += '{}h'.format(int(s/3600) % 24)
            if s > 60:
                label += '{}m'.format(int(s/60) % 60)
            label += '{}s'.format(int(s) % 60)
        return label

    def text_report(self):
        return "\n".join(itertools.chain(
            self._text_report_gen(),
            self._recommend_gen())) + "\n"

    def html_report(self):
        return crunchstat_summary.chartjs.ChartJS(self.label, [self]).html()

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
                ('Number of tasks: {}',
                 len(self.tasks),
                 None),
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
        asked_cores = self.existing_constraints.get('min_cores_per_node')
        if asked_cores is None or used_cores < asked_cores:
            yield (
                '#!! {} max CPU usage was {}% -- '
                'try runtime_constraints "min_cores_per_node":{}'
            ).format(
                self.label,
                int(math.ceil(cpu_max_rate*100)),
                int(used_cores))

    def _recommend_ram(self):
        """Recommend an economical RAM constraint for this job.

        Nodes that are advertised as "8 gibibytes" actually have what
        we might call "8 nearlygibs" of memory available for jobs.
        Here, we calculate a whole number of nearlygibs that would
        have sufficed to run the job, then recommend requesting a node
        with that number of nearlygibs (expressed as mebibytes).

        Requesting a node with "nearly 8 gibibytes" is our best hope
        of getting a node that actually has nearly 8 gibibytes
        available.  If the node manager is smart enough to account for
        the discrepancy itself when choosing/creating a node, we'll
        get an 8 GiB node with nearly 8 GiB available.  Otherwise, the
        advertised size of the next-size-smaller node (say, 6 GiB)
        will be too low to satisfy our request, so we will effectively
        get rounded up to 8 GiB.

        For example, if we need 7500 MiB, we can ask for 7500 MiB, and
        we will generally get a node that is advertised as "8 GiB" and
        has at least 7500 MiB available.  However, asking for 8192 MiB
        would either result in an unnecessarily expensive 12 GiB node
        (if node manager knows about the discrepancy), or an 8 GiB
        node which has less than 8192 MiB available and is therefore
        considered by crunch-dispatch to be too small to meet our
        constraint.

        When node manager learns how to predict the available memory
        for each node type such that crunch-dispatch always agrees
        that a node is big enough to run the job it was brought up
        for, all this will be unnecessary.  We'll just ask for exactly
        the memory we want -- even if that happens to be 8192 MiB.
        """

        used_bytes = self.stats_max['mem']['rss']
        if used_bytes == float('-Inf'):
            logger.warning('%s: no memory usage data', self.label)
            return
        used_mib = math.ceil(float(used_bytes) / 1048576)
        asked_mib = self.existing_constraints.get('min_ram_mb_per_node')

        nearlygibs = lambda mebibytes: mebibytes/AVAILABLE_RAM_RATIO/1024
        if asked_mib is None or (
                math.ceil(nearlygibs(used_mib)) < nearlygibs(asked_mib)):
            yield (
                '#!! {} max RSS was {} MiB -- '
                'try runtime_constraints "min_ram_mb_per_node":{}'
            ).format(
                self.label,
                int(used_mib),
                int(math.ceil(nearlygibs(used_mib))*AVAILABLE_RAM_RATIO*1024))

    def _format(self, val):
        """Return a string representation of a stat.

        {:.2f} for floats, default format for everything else."""
        if isinstance(val, float):
            return '{:.2f}'.format(val)
        else:
            return '{}'.format(val)


class CollectionSummarizer(Summarizer):
    def __init__(self, collection_id, **kwargs):
        logger.debug('load collection %s', collection_id)
        collection = arvados.collection.CollectionReader(collection_id)
        filenames = [filename for filename in collection]
        if len(filenames) != 1:
            raise ValueError(
                "collection {} has {} files; need exactly one".format(
                    collection_id, len(filenames)))
        super(CollectionSummarizer, self).__init__(
            collection.open(filenames[0]), **kwargs)
        self.label = collection_id


class JobSummarizer(CollectionSummarizer):
    def __init__(self, job, **kwargs):
        arv = arvados.api('v1')
        if isinstance(job, basestring):
            self.job = arv.jobs().get(uuid=job).execute()
        else:
            self.job = job
        if not self.job['log']:
            raise ValueError(
                "job {} has no log; live summary not implemented".format(
                    self.job['uuid']))
        super(JobSummarizer, self).__init__(self.job['log'], **kwargs)
        self.label = self.job['uuid']
        self.existing_constraints = self.job.get('runtime_constraints', {})


class PipelineSummarizer(object):
    def __init__(self, pipeline_instance_uuid, **kwargs):
        arv = arvados.api('v1', model=OrderedJsonModel())
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
                logger.info(
                    "%s: logdata %s", cname, component['job']['log'])
                summarizer = JobSummarizer(component['job'], **kwargs)
                summarizer.label = cname
                self.summarizers[cname] = summarizer
        self.label = pipeline_instance_uuid

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

    def html_report(self):
        return crunchstat_summary.chartjs.ChartJS(
            self.label, self.summarizers.itervalues()).html()
