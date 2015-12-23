import argparse
import gzip
import logging
import sys

from crunchstat_summary import logger, summarizer


class ArgumentParser(argparse.ArgumentParser):
    def __init__(self):
        super(ArgumentParser, self).__init__(
            description='Summarize resource usage of an Arvados Crunch job')
        src = self.add_mutually_exclusive_group()
        src.add_argument(
            '--job', type=str, metavar='UUID',
            help='Look up the specified job and read its log data from Keep')
        src.add_argument(
            '--pipeline-instance', type=str, metavar='UUID',
            help='Summarize each component of the given pipeline instance')
        src.add_argument(
            '--log-file', type=str,
            help='Read log data from a regular file')
        self.add_argument(
            '--format', type=str, choices=('html', 'text'), default='text',
            help='Report format')
        self.add_argument(
            '--verbose', action='store_true',
            help='Write progress messages to stderr')
        self.add_argument(
            '--debug', action='store_true',
            help='Write debug messages to stderr')


class Command(object):
    def __init__(self, args):
        self.args = args
        if args.debug:
            logger.setLevel(logging.DEBUG)
        elif args.verbose:
            logger.setLevel(logging.INFO)

    def run(self):
        if self.args.pipeline_instance:
            self.summer = summarizer.PipelineSummarizer(self.args.pipeline_instance)
        elif self.args.job:
            self.summer = summarizer.JobSummarizer(self.args.job)
        elif self.args.log_file:
            if self.args.log_file.endswith('.gz'):
                fh = gzip.open(self.args.log_file)
            else:
                fh = open(self.args.log_file)
            self.summer = summarizer.Summarizer(fh)
        else:
            self.summer = summarizer.Summarizer(sys.stdin)
        return self.summer.run()

    def report(self):
        if self.args.format == 'html':
            return self.summer.html_report()
        elif self.args.format == 'text':
            return self.summer.text_report()
