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
from crunchstat_summary import logger, reader

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
        s = crunchstat_summary.summarizer.Summarizer(reader.StubReader(logfile))
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
        mock_api().container_requests().list().execute.side_effect = items # parent request
        mock_api().container_requests().get().execute.return_value = self.fake_request
        mock_api().containers().get().execute.return_value = self.fake_container
        mock_cr().__iter__.return_value = [
            'crunch-run.txt', 'stderr.txt', 'node-info.txt',
            'container.json', 'crunchstat.txt', 'arv-mount.txt']
        def _open(n, mode):
            if n == "crunchstat.txt":
                return UTF8Decode(gzip.open(self.logfile))
            elif n == "arv-mount.txt":
                return UTF8Decode(gzip.open(self.arvmountlog))
            elif n == "node.json":
                return io.StringIO("{}")
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
