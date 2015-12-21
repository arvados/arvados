import argparse
import gzip
import sys

from crunchstat_summary import summarizer


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


class Command(object):
    def __init__(self, args):
        self.args = args

    def summarizer(self):
        if self.args.pipeline_instance:
            return summarizer.PipelineSummarizer(self.args.pipeline_instance)
        elif self.args.job:
            return summarizer.JobSummarizer(self.args.job)
        elif self.args.log_file:
            if self.args.log_file.endswith('.gz'):
                fh = gzip.open(self.args.log_file)
            else:
                fh = open(self.args.log_file)
            return summarizer.Summarizer(fh)
        else:
            return summarizer.Summarizer(sys.stdin)
