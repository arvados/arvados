# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import print_function

import arvados
import collections
import crunchstat_summary.dygraphs
import crunchstat_summary.reader
import datetime
import functools
import itertools
import math
import re
import sys
import threading
import _strptime

from arvados.api import OrderedJsonModel
from crunchstat_summary import logger

# Recommend memory constraints that are this multiple of an integral
# number of GiB. (Actual nodes tend to be sold in sizes like 8 GiB
# that have amounts like 7.5 GiB according to the kernel.)
AVAILABLE_RAM_RATIO = 0.95


# Workaround datetime.datetime.strptime() thread-safety bug by calling
# it once before starting threads.  https://bugs.python.org/issue7980
datetime.datetime.strptime('1999-12-31_23:59:59', '%Y-%m-%d_%H:%M:%S')


WEBCHART_CLASS = crunchstat_summary.dygraphs.DygraphsChart


class Task(object):
    def __init__(self):
        self.starttime = None
        self.series = collections.defaultdict(list)


class Summarizer(object):
    def __init__(self, logdata, label=None, skip_child_jobs=False, uuid=None, **kwargs):
        self._logdata = logdata

        self.uuid = uuid
        self.label = label
        self.starttime = None
        self.finishtime = None
        self._skip_child_jobs = skip_child_jobs

        # stats_max: {category: {stat: val}}
        self.stats_max = collections.defaultdict(
            functools.partial(collections.defaultdict, lambda: 0))
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

        logger.debug("%s: logdata %s", self.label, logdata)

    def run(self):
        logger.debug("%s: parsing logdata %s", self.label, self._logdata)
        with self._logdata as logdata:
            self._run(logdata)

    def _run(self, logdata):
        self.detected_crunch1 = False
        for line in logdata:
            if not self.detected_crunch1 and '-8i9sb-' in line:
                self.detected_crunch1 = True

            if self.detected_crunch1:
                m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) job_task (?P<task_uuid>\S+)$', line)
                if m:
                    seq = int(m.group('seq'))
                    uuid = m.group('task_uuid')
                    self.seq_to_uuid[seq] = uuid
                    logger.debug('%s: seq %d is task %s', self.label, seq, uuid)
                    continue

                m = re.search(r'^\S+ \S+ \d+ (?P<seq>\d+) (success in|failure \(#., permanent\) after) (?P<elapsed>\d+) seconds', line)
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
                    child_summarizer = ProcessSummarizer(uuid)
                    child_summarizer.stats_max = self.stats_max
                    child_summarizer.task_stats = self.task_stats
                    child_summarizer.tasks = self.tasks
                    child_summarizer.starttime = self.starttime
                    child_summarizer.run()
                    logger.debug('%s: done %s', self.label, uuid)
                    continue

                m = re.search(r'^(?P<timestamp>[^\s.]+)(\.\d+)? (?P<job_uuid>\S+) \d+ (?P<seq>\d+) stderr crunchstat: (?P<category>\S+) (?P<current>.*?)( -- interval (?P<interval>.*))?\n$', line)
                if not m:
                    continue
            else:
                # crunch2
                m = re.search(r'^(?P<timestamp>\S+) (?P<category>\S+) (?P<current>.*?)( -- interval (?P<interval>.*))?\n$', line)
                if not m:
                    continue

            if self.label is None:
                try:
                    self.label = m.group('job_uuid')
                except IndexError:
                    self.label = 'container'
            if m.group('category').endswith(':'):
                # "stderr crunchstat: notice: ..."
                continue
            elif m.group('category') in ('error', 'caught'):
                continue
            elif m.group('category') in ('read', 'open', 'cgroup', 'CID', 'Running'):
                # "stderr crunchstat: read /proc/1234/net/dev: ..."
                # (old logs are less careful with unprefixed error messages)
                continue

            if self.detected_crunch1:
                task_id = self.seq_to_uuid[int(m.group('seq'))]
            else:
                task_id = 'container'
            task = self.tasks[task_id]

            # Use the first and last crunchstat timestamps as
            # approximations of starttime and finishtime.
            timestamp = m.group('timestamp')
            if timestamp[10:11] == '_':
                timestamp = datetime.datetime.strptime(
                    timestamp, '%Y-%m-%d_%H:%M:%S')
            elif timestamp[10:11] == 'T':
                timestamp = datetime.datetime.strptime(
                    timestamp[:19], '%Y-%m-%dT%H:%M:%S')
            else:
                raise ValueError("Cannot parse timestamp {!r}".format(
                    timestamp))

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
                try:
                    for val, stat in zip(words[::2], words[1::2]):
                        if '.' in val:
                            stats[stat] = float(val)
                        else:
                            stats[stat] = int(val)
                except ValueError as e:
                    logger.warning(
                        'Error parsing value %r (stat %r, category %r): %r',
                        val, stat, category, e)
                    logger.warning('%s', line)
                    continue
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
                                    (timestamp - self.starttime, val))
                    else:
                        if stat in ['rss']:
                            task.series[category, stat].append(
                                (timestamp - self.starttime, val))
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
        if hasattr(self, 'process') and self.process['uuid'] not in label:
            label = '{} ({})'.format(label, self.process['uuid'])
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
        if not self.tasks:
            return "(no report generated)\n"
        return "\n".join(itertools.chain(
            self._text_report_gen(),
            self._recommend_gen())) + "\n"

    def html_report(self):
        return WEBCHART_CLASS(self.label, [self]).html()

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
                 self.job_tot['time']['elapsed']
                 if self.job_tot['time']['elapsed'] > 0 else 0,
                 lambda x: x * 100),
                ('Max memory used by a single task: {}GB',
                 self.stats_max['mem']['rss'],
                 lambda x: x / 1e9),
                ('Max network traffic in a single task: {}GB',
                 self.stats_max['net:eth0']['tx+rx'] +
                 self.stats_max['net:keep0']['tx+rx'],
                 lambda x: x / 1e9),
                ('Max network speed in a single interval: {}MB/s',
                 self.stats_max['net:eth0']['tx+rx__rate'] +
                 self.stats_max['net:keep0']['tx+rx__rate'],
                 lambda x: x / 1e6),
                ('Keep cache miss rate {}%',
                 (float(self.job_tot['keepcache']['miss']) /
                 float(self.job_tot['keepcalls']['get']))
                 if self.job_tot['keepcalls']['get'] > 0 else 0,
                 lambda x: x * 100.0),
                ('Keep cache utilization {}%',
                 (float(self.job_tot['blkio:0:0']['read']) /
                 float(self.job_tot['net:keep0']['rx']))
                 if self.job_tot['net:keep0']['rx'] > 0 else 0,
                 lambda x: x * 100.0)):
            format_string, val, transform = args
            if val == float('-Inf'):
                continue
            if transform:
                val = transform(val)
            yield "# "+format_string.format(self._format(val))

    def _recommend_gen(self):
        return itertools.chain(
            self._recommend_cpu(),
            self._recommend_ram(),
            self._recommend_keep_cache())

    def _recommend_cpu(self):
        """Recommend asking for 4 cores if max CPU usage was 333%"""

        constraint_key = self._map_runtime_constraint('vcpus')
        cpu_max_rate = self.stats_max['cpu']['user+sys__rate']
        if cpu_max_rate == float('-Inf'):
            logger.warning('%s: no CPU usage data', self.label)
            return
        used_cores = max(1, int(math.ceil(cpu_max_rate)))
        asked_cores = self.existing_constraints.get(constraint_key)
        if asked_cores is None or used_cores < asked_cores:
            yield (
                '#!! {} max CPU usage was {}% -- '
                'try runtime_constraints "{}":{}'
            ).format(
                self.label,
                int(math.ceil(cpu_max_rate*100)),
                constraint_key,
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

        constraint_key = self._map_runtime_constraint('ram')
        used_bytes = self.stats_max['mem']['rss']
        if used_bytes == float('-Inf'):
            logger.warning('%s: no memory usage data', self.label)
            return
        used_mib = math.ceil(float(used_bytes) / 1048576)
        asked_mib = self.existing_constraints.get(constraint_key)

        nearlygibs = lambda mebibytes: mebibytes/AVAILABLE_RAM_RATIO/1024
        if asked_mib is None or (
                math.ceil(nearlygibs(used_mib)) < nearlygibs(asked_mib)):
            yield (
                '#!! {} max RSS was {} MiB -- '
                'try runtime_constraints "{}":{}'
            ).format(
                self.label,
                int(used_mib),
                constraint_key,
                int(math.ceil(nearlygibs(used_mib))*AVAILABLE_RAM_RATIO*1024*(2**20)/self._runtime_constraint_mem_unit()))

    def _recommend_keep_cache(self):
        """Recommend increasing keep cache if utilization < 80%"""
        constraint_key = self._map_runtime_constraint('keep_cache_ram')
        if self.job_tot['net:keep0']['rx'] == 0:
            return
        utilization = (float(self.job_tot['blkio:0:0']['read']) /
                       float(self.job_tot['net:keep0']['rx']))
        asked_mib = self.existing_constraints.get(constraint_key, 256)

        if utilization < 0.8:
            yield (
                '#!! {} Keep cache utilization was {:.2f}% -- '
                'try runtime_constraints "{}":{} (or more)'
            ).format(
                self.label,
                utilization * 100.0,
                constraint_key,
                asked_mib*2*(2**20)/self._runtime_constraint_mem_unit())


    def _format(self, val):
        """Return a string representation of a stat.

        {:.2f} for floats, default format for everything else."""
        if isinstance(val, float):
            return '{:.2f}'.format(val)
        else:
            return '{}'.format(val)

    def _runtime_constraint_mem_unit(self):
        if hasattr(self, 'runtime_constraint_mem_unit'):
            return self.runtime_constraint_mem_unit
        elif self.detected_crunch1:
            return JobSummarizer.runtime_constraint_mem_unit
        else:
            return ContainerSummarizer.runtime_constraint_mem_unit

    def _map_runtime_constraint(self, key):
        if hasattr(self, 'map_runtime_constraint'):
            return self.map_runtime_constraint[key]
        elif self.detected_crunch1:
            return JobSummarizer.map_runtime_constraint[key]
        else:
            return key


class CollectionSummarizer(Summarizer):
    def __init__(self, collection_id, **kwargs):
        super(CollectionSummarizer, self).__init__(
            crunchstat_summary.reader.CollectionReader(collection_id), **kwargs)
        self.label = collection_id


def NewSummarizer(process_or_uuid, **kwargs):
    """Construct with the appropriate subclass for this uuid/object."""

    if isinstance(process_or_uuid, dict):
        process = process_or_uuid
        uuid = process['uuid']
    else:
        uuid = process_or_uuid
        process = None
        arv = arvados.api('v1', model=OrderedJsonModel())

    if '-dz642-' in uuid:
        if process is None:
            process = arv.containers().get(uuid=uuid).execute()
        klass = ContainerTreeSummarizer
    elif '-xvhdp-' in uuid:
        if process is None:
            process = arv.container_requests().get(uuid=uuid).execute()
        klass = ContainerTreeSummarizer
    elif '-8i9sb-' in uuid:
        if process is None:
            process = arv.jobs().get(uuid=uuid).execute()
        klass = JobTreeSummarizer
    elif '-d1hrv-' in uuid:
        if process is None:
            process = arv.pipeline_instances().get(uuid=uuid).execute()
        klass = PipelineSummarizer
    elif '-4zz18-' in uuid:
        return CollectionSummarizer(collection_id=uuid)
    else:
        raise ArgumentError("Unrecognized uuid %s", uuid)
    return klass(process, uuid=uuid, **kwargs)


class ProcessSummarizer(Summarizer):
    """Process is a job, pipeline, container, or container request."""

    def __init__(self, process, label=None, **kwargs):
        rdr = None
        self.process = process
        if label is None:
            label = self.process.get('name', self.process['uuid'])
        if self.process.get('log'):
            try:
                rdr = crunchstat_summary.reader.CollectionReader(self.process['log'])
            except arvados.errors.NotFoundError as e:
                logger.warning("Trying event logs after failing to read "
                               "log collection %s: %s", self.process['log'], e)
        if rdr is None:
            rdr = crunchstat_summary.reader.LiveLogReader(self.process['uuid'])
            label = label + ' (partial)'
        super(ProcessSummarizer, self).__init__(rdr, label=label, **kwargs)
        self.existing_constraints = self.process.get('runtime_constraints', {})


class JobSummarizer(ProcessSummarizer):
    runtime_constraint_mem_unit = 1048576
    map_runtime_constraint = {
        'keep_cache_ram': 'keep_cache_mb_per_task',
        'ram': 'min_ram_mb_per_node',
        'vcpus': 'min_cores_per_node',
    }


class ContainerSummarizer(ProcessSummarizer):
    runtime_constraint_mem_unit = 1


class MultiSummarizer(object):
    def __init__(self, children={}, label=None, threads=1, **kwargs):
        self.throttle = threading.Semaphore(threads)
        self.children = children
        self.label = label

    def run_and_release(self, target, *args, **kwargs):
        try:
            return target(*args, **kwargs)
        finally:
            self.throttle.release()

    def run(self):
        threads = []
        for child in self.children.itervalues():
            self.throttle.acquire()
            t = threading.Thread(target=self.run_and_release, args=(child.run, ))
            t.daemon = True
            t.start()
            threads.append(t)
        for t in threads:
            t.join()

    def text_report(self):
        txt = ''
        d = self._descendants()
        for child in d.itervalues():
            if len(d) > 1:
                txt += '### Summary for {} ({})\n'.format(
                    child.label, child.process['uuid'])
            txt += child.text_report()
            txt += '\n'
        return txt

    def _descendants(self):
        """Dict of self and all descendants.

        Nodes with nothing of their own to report (like
        MultiSummarizers) are omitted.
        """
        d = collections.OrderedDict()
        for key, child in self.children.iteritems():
            if isinstance(child, Summarizer):
                d[key] = child
            if isinstance(child, MultiSummarizer):
                d.update(child._descendants())
        return d

    def html_report(self):
        return WEBCHART_CLASS(self.label, self._descendants().itervalues()).html()


class JobTreeSummarizer(MultiSummarizer):
    """Summarizes a job and all children listed in its components field."""
    def __init__(self, job, label=None, **kwargs):
        arv = arvados.api('v1', model=OrderedJsonModel())
        label = label or job.get('name', job['uuid'])
        children = collections.OrderedDict()
        children[job['uuid']] = JobSummarizer(job, label=label, **kwargs)
        if job.get('components', None):
            preloaded = {}
            for j in arv.jobs().index(
                    limit=len(job['components']),
                    filters=[['uuid','in',job['components'].values()]]).execute()['items']:
                preloaded[j['uuid']] = j
            for cname in sorted(job['components'].keys()):
                child_uuid = job['components'][cname]
                j = (preloaded.get(child_uuid) or
                     arv.jobs().get(uuid=child_uuid).execute())
                children[child_uuid] = JobTreeSummarizer(job=j, label=cname, **kwargs)

        super(JobTreeSummarizer, self).__init__(
            children=children,
            label=label,
            **kwargs)


class PipelineSummarizer(MultiSummarizer):
    def __init__(self, instance, **kwargs):
        children = collections.OrderedDict()
        for cname, component in instance['components'].iteritems():
            if 'job' not in component:
                logger.warning(
                    "%s: skipping component with no job assigned", cname)
            else:
                logger.info(
                    "%s: job %s", cname, component['job']['uuid'])
                summarizer = JobTreeSummarizer(component['job'], label=cname, **kwargs)
                summarizer.label = '{} {}'.format(
                    cname, component['job']['uuid'])
                children[cname] = summarizer
        super(PipelineSummarizer, self).__init__(
            children=children,
            label=instance['uuid'],
            **kwargs)


class ContainerTreeSummarizer(MultiSummarizer):
    def __init__(self, root, skip_child_jobs=False, **kwargs):
        arv = arvados.api('v1', model=OrderedJsonModel())

        label = kwargs.pop('label', None) or root.get('name') or root['uuid']
        root['name'] = label

        children = collections.OrderedDict()
        todo = collections.deque((root, ))
        while len(todo) > 0:
            current = todo.popleft()
            label = current['name']
            sort_key = current['created_at']
            if current['uuid'].find('-xvhdp-') > 0:
                current = arv.containers().get(uuid=current['container_uuid']).execute()

            summer = ContainerSummarizer(current, label=label, **kwargs)
            summer.sort_key = sort_key
            children[current['uuid']] = summer

            page_filters = []
            while True:
                child_crs = arv.container_requests().index(
                    order=['uuid asc'],
                    filters=page_filters+[
                        ['requesting_container_uuid', '=', current['uuid']]],
                ).execute()
                if not child_crs['items']:
                    break
                elif skip_child_jobs:
                    logger.warning('%s: omitting stats from %d child containers'
                                   ' because --skip-child-jobs flag is on',
                                   label, child_crs['items_available'])
                    break
                page_filters = [['uuid', '>', child_crs['items'][-1]['uuid']]]
                for cr in child_crs['items']:
                    if cr['container_uuid']:
                        logger.debug('%s: container req %s', current['uuid'], cr['uuid'])
                        cr['name'] = cr.get('name') or cr['uuid']
                        todo.append(cr)
        sorted_children = collections.OrderedDict()
        for uuid in sorted(children.keys(), key=lambda uuid: children[uuid].sort_key):
            sorted_children[uuid] = children[uuid]
        super(ContainerTreeSummarizer, self).__init__(
            children=sorted_children,
            label=root['name'],
            **kwargs)
