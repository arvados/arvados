import arvados
import collections
import crunchstat_summary.command
import difflib
import glob
import gzip
import mock
import os
import unittest

TESTS_DIR = os.path.dirname(os.path.abspath(__file__))


class ReportDiff(unittest.TestCase):
    def diff_known_report(self, logfile, cmd):
        expectfile = logfile+'.report'
        expect = open(expectfile).readlines()
        self.diff_report(cmd, expect, expectfile=expectfile)

    def diff_report(self, cmd, expect, expectfile=None):
        got = [x+"\n" for x in cmd.report().strip("\n").split("\n")]
        self.assertEqual(got, expect, "\n"+"".join(difflib.context_diff(
            expect, got, fromfile=expectfile, tofile="(generated)")))


class SummarizeFile(ReportDiff):
    def test_example_files(self):
        for fnm in glob.glob(os.path.join(TESTS_DIR, '*.txt.gz')):
            logfile = os.path.join(TESTS_DIR, fnm)
            args = crunchstat_summary.command.ArgumentParser().parse_args(
                ['--log-file', logfile])
            cmd = crunchstat_summary.command.Command(args)
            cmd.run()
            self.diff_known_report(logfile, cmd)


class HTMLFromFile(ReportDiff):
    def test_example_files(self):
        # Note we don't test the output content at all yet; we're
        # mainly just verifying the --format=html option isn't ignored
        # and the HTML code path doesn't crash.
        for fnm in glob.glob(os.path.join(TESTS_DIR, '*.txt.gz')):
            logfile = os.path.join(TESTS_DIR, fnm)
            args = crunchstat_summary.command.ArgumentParser().parse_args(
                ['--format=html', '--log-file', logfile])
            cmd = crunchstat_summary.command.Command(args)
            cmd.run()
            self.assertRegexpMatches(cmd.report(), r'(?is)<html>.*</html>\s*$')


class SummarizeEdgeCases(unittest.TestCase):
    def test_error_messages(self):
        logfile = open(os.path.join(TESTS_DIR, 'crunchstat_error_messages.txt'))
        s = crunchstat_summary.summarizer.Summarizer(logfile)
        s.run()


class SummarizeJob(ReportDiff):
    fake_job_uuid = '4xphq-8i9sb-jq0ekny1xou3zoh'
    fake_log_id = 'fake-log-collection-id'
    fake_job = {
        'uuid': fake_job_uuid,
        'log': fake_log_id,
    }
    logfile = os.path.join(TESTS_DIR, 'logfile_20151204190335.txt.gz')

    @mock.patch('arvados.collection.CollectionReader')
    @mock.patch('arvados.api')
    def test_job_report(self, mock_api, mock_cr):
        mock_api().jobs().get().execute.return_value = self.fake_job
        mock_cr().__iter__.return_value = ['fake-logfile.txt']
        mock_cr().open.return_value = gzip.open(self.logfile)
        args = crunchstat_summary.command.ArgumentParser().parse_args(
            ['--job', self.fake_job_uuid])
        cmd = crunchstat_summary.command.Command(args)
        cmd.run()
        self.diff_known_report(self.logfile, cmd)
        mock_api().jobs().get.assert_called_with(uuid=self.fake_job_uuid)
        mock_cr.assert_called_with(self.fake_log_id)
        mock_cr().open.assert_called_with('fake-logfile.txt')


class SummarizePipeline(ReportDiff):
    fake_instance = {
        'uuid': 'zzzzz-d1hrv-i3e77t9z5y8j9cc',
        'owner_uuid': 'zzzzz-tpzed-xurymjxw79nv3jz',
        'components': collections.OrderedDict([
            ['foo', {
                'job': {
                    'uuid': 'zzzzz-8i9sb-000000000000000',
                    'log': 'fake-log-pdh-0',
                    'runtime_constraints': {
                        'min_ram_mb_per_node': 900,
                        'min_cores_per_node': 1,
                    },
                },
            }],
            ['bar', {
                'job': {
                    'uuid': 'zzzzz-8i9sb-000000000000001',
                    'log': 'fake-log-pdh-1',
                    'runtime_constraints': {
                        'min_ram_mb_per_node': 900,
                        'min_cores_per_node': 1,
                    },
                },
            }],
            ['no-job-assigned', {}],
            ['unfinished-job', {
                'job': {
                    'uuid': 'zzzzz-8i9sb-xxxxxxxxxxxxxxx',
                },
            }],
            ['baz', {
                'job': {
                    'uuid': 'zzzzz-8i9sb-000000000000002',
                    'log': 'fake-log-pdh-2',
                    'runtime_constraints': {
                        'min_ram_mb_per_node': 900,
                        'min_cores_per_node': 1,
                    },
                },
            }]]),
    }

    @mock.patch('arvados.collection.CollectionReader')
    @mock.patch('arvados.api')
    def test_pipeline(self, mock_api, mock_cr):
        logfile = os.path.join(TESTS_DIR, 'logfile_20151204190335.txt.gz')
        mock_api().pipeline_instances().get().execute. \
            return_value = self.fake_instance
        mock_cr().__iter__.return_value = ['fake-logfile.txt']
        mock_cr().open.side_effect = [gzip.open(logfile) for _ in range(3)]
        args = crunchstat_summary.command.ArgumentParser().parse_args(
            ['--pipeline-instance', self.fake_instance['uuid']])
        cmd = crunchstat_summary.command.Command(args)
        cmd.run()

        job_report = [
            line for line in open(logfile+'.report').readlines()
            if not line.startswith('#!! ')]
        expect = (
            ['### Summary for foo (zzzzz-8i9sb-000000000000000)\n'] +
            job_report + ['\n'] +
            ['### Summary for bar (zzzzz-8i9sb-000000000000001)\n'] +
            job_report + ['\n'] +
            ['### Summary for baz (zzzzz-8i9sb-000000000000002)\n'] +
            job_report)
        self.diff_report(cmd, expect)
        mock_cr.assert_has_calls(
            [
                mock.call('fake-log-pdh-0'),
                mock.call('fake-log-pdh-1'),
                mock.call('fake-log-pdh-2'),
            ], any_order=True)
        mock_cr().open.assert_called_with('fake-logfile.txt')
