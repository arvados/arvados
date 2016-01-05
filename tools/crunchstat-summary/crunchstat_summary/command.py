import argparse


class ArgumentParser(argparse.ArgumentParser):
    def __init__(self):
        super(ArgumentParser, self).__init__(
            description='Summarize resource usage of an Arvados Crunch job')
        src = self.add_mutually_exclusive_group()
        src.add_argument(
            '--job', type=str, metavar='UUID',
            help='Look up the specified job and read its log data from Keep')
        src.add_argument(
            '--log-file', type=str,
            help='Read log data from a regular file')
