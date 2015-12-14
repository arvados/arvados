import crunchstat_summary.command
import crunchstat_summary.summarizer
import difflib
import glob
import os
import unittest


class ExampleLogsTestCase(unittest.TestCase):
    def test_example_files(self):
        dirname = os.path.dirname(os.path.abspath(__file__))
        for fnm in glob.glob(os.path.join(dirname, '*.txt.gz')):
            logfile = os.path.join(dirname, fnm)
            args = crunchstat_summary.command.ArgumentParser().parse_args(
                ['--log-file', logfile])
            summarizer = crunchstat_summary.summarizer.Summarizer(args)
            summarizer.run()
            got = [x+"\n" for x in summarizer.report().strip("\n").split("\n")]
            expectfile = logfile+'.report'
            expect = open(expectfile).readlines()
            self.assertEqual(got, expect, "\n"+"".join(difflib.context_diff(
                expect, got, fromfile=expectfile, tofile="(generated)")))
