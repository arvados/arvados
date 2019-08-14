# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import argparse
import gzip
from io import open
import logging
import sys

from crunchstat_summary import logger, summarizer


class ArgumentParser(argparse.ArgumentParser):
    def __init__(self):
        super(ArgumentParser, self).__init__(
            description='Summarize resource usage of an Arvados Crunch job')
        src = self.add_mutually_exclusive_group()
        src.add_argument(
            '--job', '--container-request',
            type=str, metavar='UUID',
            help='Look up the specified job or container request '
            'and read its log data from Keep (or from the Arvados event log, '
            'if the job is still running)')
        src.add_argument(
            '--container',
            type=str, metavar='UUID',
            help='[Deprecated] Look up the specified container find its container request '
            'and read its log data from Keep (or from the Arvados event log, '
            'if the job is still running)')
        src.add_argument(
            '--pipeline-instance', type=str, metavar='UUID',
            help='[Deprecated] Summarize each component of the given pipeline instance (historical pre-1.4)')
        src.add_argument(
            '--log-file', type=str,
            help='Read log data from a regular file')
        self.add_argument(
            '--skip-child-jobs', action='store_true',
            help='Do not include stats from child jobs/containers')
        self.add_argument(
            '--format', type=str, choices=('html', 'text'), default='text',
            help='Report format')
        self.add_argument(
            '--threads', type=int, default=8,
            help='Maximum worker threads to run')
        self.add_argument(
            '--verbose', '-v', action='count', default=0,
            help='Log more information (once for progress, twice for debug)')


class UTF8Decode(object):
    '''Wrap a file-like iterable to decode UTF-8 bytes into a strings
    '''
    def __init__(self, fh):
        self.fh = fh

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

    def __iter__(self):
        return self

    def __next__(self):
        return next(self.fh).decode('utf-8')

    next = __next__

    def close(self):
        # mimic Gzip behavior and don't close underlying object
        pass


class Command(object):
    def __init__(self, args):
        self.args = args
        logger.setLevel(logging.WARNING - 10 * args.verbose)

    def run(self):
        kwargs = {
            'skip_child_jobs': self.args.skip_child_jobs,
            'threads': self.args.threads,
        }
        if self.args.pipeline_instance:
            self.summer = summarizer.NewSummarizer(self.args.pipeline_instance, **kwargs)
        elif self.args.job:
            self.summer = summarizer.NewSummarizer(self.args.job, **kwargs)
        elif self.args.container:
            self.summer = summarizer.NewSummarizer(self.args.container, **kwargs)
        elif self.args.log_file:
            if self.args.log_file.endswith('.gz'):
                fh = UTF8Decode(gzip.open(self.args.log_file))
            else:
                fh = open(self.args.log_file, mode = 'r', encoding = 'utf-8')
            self.summer = summarizer.Summarizer(fh, **kwargs)
        else:
            self.summer = summarizer.Summarizer(sys.stdin, **kwargs)
        return self.summer.run()

    def report(self):
        if self.args.format == 'html':
            return self.summer.html_report()
        elif self.args.format == 'text':
            return self.summer.text_report()
