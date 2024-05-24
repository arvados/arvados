# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import itertools
import os
import stat
import subprocess
import unittest

import parameterized
import pytest
from pathlib import Path
from unittest import mock

import arvados
import arvados.util

class MkdirDashPTest(unittest.TestCase):
    def setUp(self):
        try:
            os.path.mkdir('./tmp')
        except:
            pass
    def tearDown(self):
        try:
            os.unlink('./tmp/bar')
            os.rmdir('./tmp/foo')
            os.rmdir('./tmp')
        except:
            pass
    def runTest(self):
        arvados.util.mkdir_dash_p('./tmp/foo')
        with open('./tmp/bar', 'wb') as f:
            f.write(b'bar')
        self.assertRaises(OSError, arvados.util.mkdir_dash_p, './tmp/bar')


class RunCommandTestCase(unittest.TestCase):
    def test_success(self):
        stdout, stderr = arvados.util.run_command(['echo', 'test'],
                                                  stderr=subprocess.PIPE)
        self.assertEqual("test\n".encode(), stdout)
        self.assertEqual("".encode(), stderr)

    def test_failure(self):
        with self.assertRaises(arvados.errors.CommandFailedError):
            arvados.util.run_command(['false'])

class KeysetTestHelper:
    def __init__(self, expect):
        self.n = 0
        self.expect = expect

    def fn(self, **kwargs):
        if self.expect[self.n][0] != kwargs:
            raise Exception("Didn't match %s != %s" % (self.expect[self.n][0], kwargs))
        return self

    def execute(self, num_retries):
        self.n += 1
        return self.expect[self.n-1][1]

_SELECT_FAKE_ITEM = {
    'uuid': 'zzzzz-zyyyz-zzzzzyyyyywwwww',
    'name': 'KeysetListAllTestCase.test_select mock',
    'created_at': '2023-08-28T12:34:56.123456Z',
}

class KeysetListAllTestCase(unittest.TestCase):
    def test_empty(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [])

    def test_oneitem(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", "=", "1"], ["uuid", ">", "1"]]},
            {"items": []}
        ],[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">", "1"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"}])

    def test_onepage2(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "2"], ["uuid", "!=", "2"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}])

    def test_onepage3(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}, {"created_at": "3", "uuid": "3"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "3"], ["uuid", "!=", "3"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}, {"created_at": "3", "uuid": "3"}])


    def test_twopage(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "2"], ["uuid", "!=", "2"]]},
            {"items": [{"created_at": "3", "uuid": "3"}, {"created_at": "4", "uuid": "4"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "4"], ["uuid", "!=", "4"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"},
                              {"created_at": "2", "uuid": "2"},
                              {"created_at": "3", "uuid": "3"},
                              {"created_at": "4", "uuid": "4"}
        ])

    def test_repeated_key(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}, {"created_at": "2", "uuid": "3"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "2"], ["uuid", "!=", "3"]]},
            {"items": [{"created_at": "2", "uuid": "2"}, {"created_at": "2", "uuid": "4"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", "=", "2"], ["uuid", ">", "4"]]},
            {"items": []}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">", "2"]]},
            {"items": [{"created_at": "3", "uuid": "5"}, {"created_at": "4", "uuid": "6"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "4"], ["uuid", "!=", "6"]]},
            {"items": []}
        ],
        ])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"},
                              {"created_at": "2", "uuid": "2"},
                              {"created_at": "2", "uuid": "3"},
                              {"created_at": "2", "uuid": "4"},
                              {"created_at": "3", "uuid": "5"},
                              {"created_at": "4", "uuid": "6"}
        ])

    def test_onepage_withfilter(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["foo", ">", "bar"]]},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "2"], ["uuid", "!=", "2"], ["foo", ">", "bar"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn, filters=[["foo", ">", "bar"]]))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}])

    def test_onepage_desc(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at desc", "uuid desc"], "filters": []},
            {"items": [{"created_at": "2", "uuid": "2"}, {"created_at": "1", "uuid": "1"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at desc", "uuid desc"], "filters": [["created_at", "<=", "1"], ["uuid", "!=", "1"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn, ascending=False))
        self.assertEqual(ls, [{"created_at": "2", "uuid": "2"}, {"created_at": "1", "uuid": "1"}])

    @parameterized.parameterized.expand(zip(
        itertools.cycle(_SELECT_FAKE_ITEM),
        itertools.chain.from_iterable(
            itertools.combinations(_SELECT_FAKE_ITEM, count)
            for count in range(len(_SELECT_FAKE_ITEM) + 1)
        ),
    ))
    def test_select(self, order_key, select):
        # keyset_list_all must have both uuid and order_key to function.
        # Test that it selects those fields along with user-specified ones.
        expect_select = {'uuid', order_key, *select}
        item = {
            key: value
            for key, value in _SELECT_FAKE_ITEM.items()
            if key in expect_select
        }
        list_func = mock.Mock()
        list_func().execute = mock.Mock(
            side_effect=[
                {'items': [item]},
                {'items': []},
                {'items': []},
            ],
        )
        list_func.reset_mock()
        actual = list(arvados.util.keyset_list_all(list_func, order_key, select=list(select)))
        self.assertEqual(actual, [item])
        calls = list_func.call_args_list
        self.assertTrue(len(calls) >= 2, "list_func() not called enough to exhaust items")
        for args, kwargs in calls:
            self.assertEqual(set(kwargs.get('select', ())), expect_select)


class TestBaseDirectories:
    SELF_PATH = Path(__file__)

    @pytest.fixture
    def dir_spec(self, tmp_path):
        return arvados.util._BaseDirectorySpec(
            'TEST_DIRECTORY',
            'XDG_TEST_HOME',
            Path('.test'),
            'XDG_TEST_DIRS',
            f"{tmp_path / '.test1'}:{tmp_path / '.test2'}",
        )

    @pytest.fixture
    def env(self, tmp_path):
        return {'HOME': str(tmp_path)}

    @pytest.fixture
    def umask(self):
        orig_umask = os.umask(0o002)
        try:
            yield
        finally:
            os.umask(orig_umask)

    def test_search_systemd_dirs(self, dir_spec, env, tmp_path):
        env['TEST_DIRECTORY'] = f'{tmp_path}:{self.SELF_PATH.parent}'
        dirs = arvados.util._BaseDirectories(dir_spec, env, 'tests')
        actual = list(dirs.search(self.SELF_PATH.name))
        assert actual == [self.SELF_PATH]

    def test_search_xdg_home(self, dir_spec, env, tmp_path):
        env['XDG_TEST_HOME'] = str(self.SELF_PATH.parent.parent)
        dirs = arvados.util._BaseDirectories(dir_spec, env, 'tests')
        actual = list(dirs.search(self.SELF_PATH.name))
        assert actual == [self.SELF_PATH]

    def test_search_xdg_dirs(self, dir_spec, env, tmp_path):
        env['XDG_TEST_DIRS'] = f'{tmp_path}:{self.SELF_PATH.parent.parent}'
        dirs = arvados.util._BaseDirectories(dir_spec, env, 'tests')
        actual = list(dirs.search(self.SELF_PATH.name))
        assert actual == [self.SELF_PATH]

    def test_search_all_dirs(self, dir_spec, env, tmp_path):
        env['TEST_DIRECTORY'] = f'{tmp_path}:{self.SELF_PATH.parent}'
        env['XDG_TEST_HOME'] = str(self.SELF_PATH.parent.parent)
        env['XDG_TEST_DIRS'] = f'{tmp_path}:{self.SELF_PATH.parent.parent}'
        dirs = arvados.util._BaseDirectories(dir_spec, env, 'tests')
        actual = list(dirs.search(self.SELF_PATH.name))
        assert actual == [self.SELF_PATH, self.SELF_PATH, self.SELF_PATH]

    def test_search_default_home(self, dir_spec, env, tmp_path):
        expected = tmp_path / dir_spec.xdg_home_default / 'default_home'
        expected.parent.mkdir()
        expected.touch()
        dirs = arvados.util._BaseDirectories(dir_spec, env, '.')
        actual = list(dirs.search(expected.name))
        assert actual == [expected]

    def test_search_default_dirs(self, dir_spec, env, tmp_path):
        _, _, default_dir = dir_spec.xdg_dirs_default.rpartition(':')
        expected = Path(default_dir, 'default_dirs')
        expected.parent.mkdir()
        expected.touch()
        dirs = arvados.util._BaseDirectories(dir_spec, env, '.')
        actual = list(dirs.search(expected.name))
        assert actual == [expected]

    def test_search_no_default_dirs(self, dir_spec, env, tmp_path):
        dir_spec.xdg_dirs_key = None
        dir_spec.xdg_dirs_default = None
        for subdir in ['.test1', '.test2', dir_spec.xdg_home_default]:
            expected = tmp_path / subdir / 'no_dirs'
            expected.parent.mkdir()
            expected.touch()
        dirs = arvados.util._BaseDirectories(dir_spec, env, '.')
        actual = list(dirs.search(expected.name))
        assert actual == [expected]

    def test_ignore_relative_directories(self, dir_spec, env, tmp_path):
        test_path = Path(*self.SELF_PATH.parts[-2:])
        assert test_path.exists(), "test setup problem: need an existing file in a subdirectory of ."
        parent_path = str(test_path.parent)
        env['TEST_DIRECTORY'] = '.'
        env['XDG_TEST_HOME'] = parent_path
        env['XDG_TEST_DIRS'] = parent_path
        dirs = arvados.util._BaseDirectories(dir_spec, env, parent_path)
        assert not list(dirs.search(test_path.name))

    def test_storage_path_systemd(self, dir_spec, env, tmp_path):
        expected = tmp_path / 'rwsystemd'
        expected.mkdir(0o700)
        env['TEST_DIRECTORY'] = str(expected)
        dirs = arvados.util._BaseDirectories(dir_spec, env)
        assert dirs.storage_path() == expected

    def test_storage_path_systemd_mixed_modes(self, dir_spec, env, tmp_path):
        rodir = tmp_path / 'rodir'
        rodir.mkdir(0o500)
        expected = tmp_path / 'rwdir'
        expected.mkdir(0o700)
        env['TEST_DIRECTORY'] = f'{rodir}:{expected}'
        dirs = arvados.util._BaseDirectories(dir_spec, env)
        assert dirs.storage_path() == expected

    def test_storage_path_xdg_home(self, dir_spec, env, tmp_path):
        expected = tmp_path / '.xdghome' / 'arvados'
        env['XDG_TEST_HOME'] = str(expected.parent)
        dirs = arvados.util._BaseDirectories(dir_spec, env)
        assert dirs.storage_path() == expected
        exp_mode = stat.S_IFDIR | stat.S_IWUSR
        assert (expected.stat().st_mode & exp_mode) == exp_mode

    def test_storage_path_default(self, dir_spec, env, tmp_path):
        expected = tmp_path / dir_spec.xdg_home_default / 'arvados'
        dirs = arvados.util._BaseDirectories(dir_spec, env)
        assert dirs.storage_path() == expected
        exp_mode = stat.S_IFDIR | stat.S_IWUSR
        assert (expected.stat().st_mode & exp_mode) == exp_mode

    @pytest.mark.parametrize('subdir,mode', [
        ('str/dir', 0o750),
        (Path('sub', 'path'), 0o770),
    ])
    def test_storage_path_subdir(self, dir_spec, env, umask, tmp_path, subdir, mode):
        expected = tmp_path / dir_spec.xdg_home_default / 'arvados' / subdir
        dirs = arvados.util._BaseDirectories(dir_spec, env)
        actual = dirs.storage_path(subdir, mode)
        assert actual == expected
        expect_mode = mode | stat.S_IFDIR
        actual_mode = actual.stat().st_mode
        assert (actual_mode & expect_mode) == expect_mode
        assert not (actual_mode & stat.S_IRWXO)

    def test_empty_xdg_home(self, dir_spec, env, tmp_path):
        env['XDG_TEST_HOME'] = ''
        expected = tmp_path / dir_spec.xdg_home_default / 'emptyhome'
        dirs = arvados.util._BaseDirectories(dir_spec, env, expected.name)
        assert dirs.storage_path() == expected

    def test_empty_xdg_dirs(self, dir_spec, env, tmp_path):
        env['XDG_TEST_DIRS'] = ''
        _, _, default_dir = dir_spec.xdg_dirs_default.rpartition(':')
        expected = Path(default_dir, 'empty_dirs')
        expected.parent.mkdir()
        expected.touch()
        dirs = arvados.util._BaseDirectories(dir_spec, env, '.')
        actual = list(dirs.search(expected.name))
        assert actual == [expected]

    def test_spec_key_lookup(self):
        dirs = arvados.util._BaseDirectories('CACHE')
        assert dirs._spec.systemd_key == 'CACHE_DIRECTORY'
        assert dirs._spec.xdg_dirs_key is None

    def test_spec_enum_lookup(self):
        dirs = arvados.util._BaseDirectories(arvados.util._BaseDirectorySpecs.CONFIG)
        assert dirs._spec.systemd_key == 'CONFIGURATION_DIRECTORY'
