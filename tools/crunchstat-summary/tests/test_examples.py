# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import arvados
import collections
import crunchstat_summary.command
import difflib
import glob
import gzip
import io
import logging
import mock
import os
import sys
import unittest

from crunchstat_summary.command import UTF8Decode
from crunchstat_summary import logger

TESTS_DIR = os.path.dirname(os.path.abspath(__file__))


class TestCase(unittest.TestCase):
    def setUp(self):
        self.logbuf = io.StringIO()
        self.loghandler = logging.StreamHandler(stream=self.logbuf)
        logger.addHandler(self.loghandler)
        logger.setLevel(logging.WARNING)

    def tearDown(self):
        logger.removeHandler(self.loghandler)

    def diff_known_report(self, logfile, cmd):
        expectfile = logfile+'.report'
        with io.open(expectfile, encoding='utf-8') as f:
            expect = f.readlines()
        self.diff_report(cmd, expect, expectfile=expectfile)

    def diff_report(self, cmd, expect, expectfile='(expected)'):
        got = [x+"\n" for x in cmd.report().strip("\n").split("\n")]
        self.assertEqual(got, expect, "\n"+"".join(difflib.context_diff(
            expect, got, fromfile=expectfile, tofile="(generated)")))


class SummarizeFile(TestCase):
    def test_example_files(self):
        for fnm in glob.glob(os.path.join(TESTS_DIR, '*.txt.gz')):
            logfile = os.path.join(TESTS_DIR, fnm)
            args = crunchstat_summary.command.ArgumentParser().parse_args(
                ['--log-file', logfile])
            cmd = crunchstat_summary.command.Command(args)
            cmd.run()
            self.diff_known_report(logfile, cmd)


class HTMLFromFile(TestCase):
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
            self.assertRegex(cmd.report(), r'(?is)<html>.*</html>\s*$')


class SummarizeEdgeCases(TestCase):
    def test_error_messages(self):
        logfile = io.open(os.path.join(TESTS_DIR, 'crunchstat_error_messages.txt'), encoding='utf-8')
        s = crunchstat_summary.summarizer.Summarizer(logfile)
        s.run()
        self.assertRegex(self.logbuf.getvalue(), r'CPU stats are missing -- possible cluster configuration issue')
        self.assertRegex(self.logbuf.getvalue(), r'memory stats are missing -- possible cluster configuration issue')
        self.assertRegex(self.logbuf.getvalue(), r'network I/O stats are missing -- possible cluster configuration issue')
        self.assertRegex(self.logbuf.getvalue(), r'storage space stats are missing -- possible cluster configuration issue')


class SummarizeContainerCommon(TestCase):
    fake_container = {
        'uuid': '9tee4-dz642-lymtndkpy39eibk',
        'created_at': '2017-08-18T14:27:25.371388141',
        'log': '9tee4-4zz18-ihyzym9tcwjwg4r',
    }
    fake_request = {
        'uuid': '9tee4-xvhdp-kk0ja1cl8b2kr1y',
        'name': 'container',
        'created_at': '2017-08-18T14:27:25.242339223Z',
        'container_uuid': fake_container['uuid'],
        'runtime_constraints': {
            'vcpus': 1,
            'ram': 2621440000
            },
        'log_uuid' : '9tee4-4zz18-m2swj50nk0r8b6y'
        }

    logfile = os.path.join(
        TESTS_DIR, 'container_request_9tee4-xvhdp-kk0ja1cl8b2kr1y-crunchstat.txt.gz')
    arvmountlog = os.path.join(
        TESTS_DIR, 'container_request_9tee4-xvhdp-kk0ja1cl8b2kr1y-arv-mount.txt.gz')

    @mock.patch('arvados.collection.CollectionReader')
    @mock.patch('arvados.api')
    def check_common(self, mock_api, mock_cr):
        items = [ {'items':[self.fake_request]}] + [{'items':[]}] * 100
        # Index and list mean the same thing, but are used in different places in the
        # code. It's fragile, but exploit that fact to distinguish the two uses.
        mock_api().container_requests().index().execute.return_value = {'items': [] }  # child_crs
        mock_api().container_requests().list().execute.side_effect = items # parent request
        mock_api().container_requests().get().execute.return_value = self.fake_request
        mock_api().containers().get().execute.return_value = self.fake_container
        mock_cr().__iter__.return_value = [
            'crunch-run.txt', 'stderr.txt', 'node-info.txt',
            'container.json', 'crunchstat.txt', 'arv-mount.txt']
        def _open(n):
            if n == "crunchstat.txt":
                return UTF8Decode(gzip.open(self.logfile))
            elif n == "arv-mount.txt":
                return UTF8Decode(gzip.open(self.arvmountlog))
        mock_cr().open.side_effect = _open
        args = crunchstat_summary.command.ArgumentParser().parse_args(
            self.arg_strings)
        cmd = crunchstat_summary.command.Command(args)
        cmd.run()
        self.diff_known_report(self.reportfile, cmd)



class SummarizeContainer(SummarizeContainerCommon):
    uuid = '9tee4-dz642-lymtndkpy39eibk'
    reportfile = os.path.join(TESTS_DIR, 'container_%s.txt.gz' % uuid)
    arg_strings = ['--container', uuid, '-v', '-v']

    def test_container(self):
        self.check_common()


class SummarizeContainerRequest(SummarizeContainerCommon):
    uuid = '9tee4-xvhdp-kk0ja1cl8b2kr1y'
    reportfile = os.path.join(TESTS_DIR, 'container_request_%s.txt.gz' % uuid)
    arg_strings = ['--container-request', uuid, '-v', '-v']

    def test_container_request(self):
        self.check_common()
        self.assertNotRegex(self.logbuf.getvalue(), r'stats are missing')
        self.assertNotRegex(self.logbuf.getvalue(), r'possible cluster configuration issue')


class SummarizeJob(TestCase):
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
        mock_cr().open.return_value = UTF8Decode(gzip.open(self.logfile))
        args = crunchstat_summary.command.ArgumentParser().parse_args(
            ['--job', self.fake_job_uuid])
        cmd = crunchstat_summary.command.Command(args)
        cmd.run()
        self.diff_known_report(self.logfile, cmd)
        mock_api().jobs().get.assert_called_with(uuid=self.fake_job_uuid)
        mock_cr.assert_called_with(self.fake_log_id)
        mock_cr().open.assert_called_with('fake-logfile.txt')


class SummarizePipeline(TestCase):
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
        mock_cr().open.side_effect = [UTF8Decode(gzip.open(logfile)) for _ in range(3)]
        args = crunchstat_summary.command.ArgumentParser().parse_args(
            ['--pipeline-instance', self.fake_instance['uuid']])
        cmd = crunchstat_summary.command.Command(args)
        cmd.run()

        with io.open(logfile+'.report', encoding='utf-8') as f:
            job_report = [line for line in f if not line.startswith('#!! ')]
        expect = (
            ['### Summary for foo (zzzzz-8i9sb-000000000000000)\n'] +
            job_report + ['\n'] +
            ['### Summary for bar (zzzzz-8i9sb-000000000000001)\n'] +
            job_report + ['\n'] +
            ['### Summary for unfinished-job (partial) (zzzzz-8i9sb-xxxxxxxxxxxxxxx)\n',
             '(no report generated)\n',
             '\n'] +
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


class SummarizeACRJob(TestCase):
    fake_job = {
        'uuid': 'zzzzz-8i9sb-i3e77t9z5y8j9cc',
        'owner_uuid': 'zzzzz-tpzed-xurymjxw79nv3jz',
        'components': {
            'foo': 'zzzzz-8i9sb-000000000000000',
            'bar': 'zzzzz-8i9sb-000000000000001',
            'unfinished-job': 'zzzzz-8i9sb-xxxxxxxxxxxxxxx',
            'baz': 'zzzzz-8i9sb-000000000000002',
        }
    }
    fake_jobs_index = { 'items': [
        {
            'uuid': 'zzzzz-8i9sb-000000000000000',
            'log': 'fake-log-pdh-0',
            'runtime_constraints': {
                'min_ram_mb_per_node': 900,
                'min_cores_per_node': 1,
            },
        },
        {
            'uuid': 'zzzzz-8i9sb-000000000000001',
            'log': 'fake-log-pdh-1',
            'runtime_constraints': {
                'min_ram_mb_per_node': 900,
                'min_cores_per_node': 1,
            },
        },
        {
            'uuid': 'zzzzz-8i9sb-xxxxxxxxxxxxxxx',
        },
        {
            'uuid': 'zzzzz-8i9sb-000000000000002',
            'log': 'fake-log-pdh-2',
            'runtime_constraints': {
                'min_ram_mb_per_node': 900,
                'min_cores_per_node': 1,
            },
        },
    ]}
    @mock.patch('arvados.collection.CollectionReader')
    @mock.patch('arvados.api')
    def test_acr_job(self, mock_api, mock_cr):
        logfile = os.path.join(TESTS_DIR, 'logfile_20151204190335.txt.gz')
        mock_api().jobs().index().execute.return_value = self.fake_jobs_index
        mock_api().jobs().get().execute.return_value = self.fake_job
        mock_cr().__iter__.return_value = ['fake-logfile.txt']
        mock_cr().open.side_effect = [UTF8Decode(gzip.open(logfile)) for _ in range(3)]
        args = crunchstat_summary.command.ArgumentParser().parse_args(
            ['--job', self.fake_job['uuid']])
        cmd = crunchstat_summary.command.Command(args)
        cmd.run()

        with io.open(logfile+'.report', encoding='utf-8') as f:
            job_report = [line for line in f if not line.startswith('#!! ')]
        expect = (
            ['### Summary for zzzzz-8i9sb-i3e77t9z5y8j9cc (partial) (zzzzz-8i9sb-i3e77t9z5y8j9cc)\n',
             '(no report generated)\n',
             '\n'] +
            ['### Summary for bar (zzzzz-8i9sb-000000000000001)\n'] +
            job_report + ['\n'] +
            ['### Summary for baz (zzzzz-8i9sb-000000000000002)\n'] +
            job_report + ['\n'] +
            ['### Summary for foo (zzzzz-8i9sb-000000000000000)\n'] +
            job_report + ['\n'] +
            ['### Summary for unfinished-job (partial) (zzzzz-8i9sb-xxxxxxxxxxxxxxx)\n',
             '(no report generated)\n']
        )
        self.diff_report(cmd, expect)
        mock_cr.assert_has_calls(
            [
                mock.call('fake-log-pdh-0'),
                mock.call('fake-log-pdh-1'),
                mock.call('fake-log-pdh-2'),
            ], any_order=True)
        mock_cr().open.assert_called_with('fake-logfile.txt')
