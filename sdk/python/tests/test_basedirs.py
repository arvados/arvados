# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import stat

import pytest

from pathlib import Path

from arvados._internal import basedirs

class TestBaseDirectories:
    SELF_PATH = Path(__file__)

    @pytest.fixture
    def dir_spec(self, tmp_path):
        return basedirs.BaseDirectorySpec(
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
        dirs = basedirs.BaseDirectories(dir_spec, env, 'tests')
        actual = list(dirs.search(self.SELF_PATH.name))
        assert actual == [self.SELF_PATH]

    def test_search_xdg_home(self, dir_spec, env, tmp_path):
        env['XDG_TEST_HOME'] = str(self.SELF_PATH.parent.parent)
        dirs = basedirs.BaseDirectories(dir_spec, env, 'tests')
        actual = list(dirs.search(self.SELF_PATH.name))
        assert actual == [self.SELF_PATH]

    def test_search_xdg_dirs(self, dir_spec, env, tmp_path):
        env['XDG_TEST_DIRS'] = f'{tmp_path}:{self.SELF_PATH.parent.parent}'
        dirs = basedirs.BaseDirectories(dir_spec, env, 'tests')
        actual = list(dirs.search(self.SELF_PATH.name))
        assert actual == [self.SELF_PATH]

    def test_search_all_dirs(self, dir_spec, env, tmp_path):
        env['TEST_DIRECTORY'] = f'{tmp_path}:{self.SELF_PATH.parent}'
        env['XDG_TEST_HOME'] = str(self.SELF_PATH.parent.parent)
        env['XDG_TEST_DIRS'] = f'{tmp_path}:{self.SELF_PATH.parent.parent}'
        dirs = basedirs.BaseDirectories(dir_spec, env, 'tests')
        actual = list(dirs.search(self.SELF_PATH.name))
        assert actual == [self.SELF_PATH, self.SELF_PATH, self.SELF_PATH]

    def test_search_default_home(self, dir_spec, env, tmp_path):
        expected = tmp_path / dir_spec.xdg_home_default / 'default_home'
        expected.parent.mkdir()
        expected.touch()
        dirs = basedirs.BaseDirectories(dir_spec, env, '.')
        actual = list(dirs.search(expected.name))
        assert actual == [expected]

    def test_search_default_dirs(self, dir_spec, env, tmp_path):
        _, _, default_dir = dir_spec.xdg_dirs_default.rpartition(':')
        expected = Path(default_dir, 'default_dirs')
        expected.parent.mkdir()
        expected.touch()
        dirs = basedirs.BaseDirectories(dir_spec, env, '.')
        actual = list(dirs.search(expected.name))
        assert actual == [expected]

    def test_search_no_default_dirs(self, dir_spec, env, tmp_path):
        dir_spec.xdg_dirs_key = None
        dir_spec.xdg_dirs_default = None
        for subdir in ['.test1', '.test2', dir_spec.xdg_home_default]:
            expected = tmp_path / subdir / 'no_dirs'
            expected.parent.mkdir()
            expected.touch()
        dirs = basedirs.BaseDirectories(dir_spec, env, '.')
        actual = list(dirs.search(expected.name))
        assert actual == [expected]

    def test_ignore_relative_directories(self, dir_spec, env, tmp_path):
        test_path = Path(*self.SELF_PATH.parts[-2:])
        assert test_path.exists(), "test setup problem: need an existing file in a subdirectory of ."
        parent_path = str(test_path.parent)
        env['TEST_DIRECTORY'] = '.'
        env['XDG_TEST_HOME'] = parent_path
        env['XDG_TEST_DIRS'] = parent_path
        dirs = basedirs.BaseDirectories(dir_spec, env, parent_path)
        assert not list(dirs.search(test_path.name))

    def test_search_warns_nondefault_home(self, dir_spec, env, tmp_path, caplog):
        search_path = tmp_path / dir_spec.xdg_home_default / 'Search' / 'SearchConfig'
        search_path.parent.mkdir(parents=True)
        search_path.touch()
        env[dir_spec.xdg_home_key] = str(tmp_path / '.nonexistent')
        dirs = basedirs.BaseDirectories(dir_spec, env, search_path.parent.name)
        results = list(dirs.search(search_path.name))
        expect_msg = "{} was not found under your configured ${} ({}), but does exist at the default location ({})".format(
            Path(*search_path.parts[-2:]),
            dir_spec.xdg_home_key,
            env[dir_spec.xdg_home_key],
            Path(*search_path.parts[:-2]),
        )
        assert caplog.messages
        assert any(msg.startswith(expect_msg) for msg in caplog.messages)
        assert not results

    def test_storage_path_systemd(self, dir_spec, env, tmp_path):
        expected = tmp_path / 'rwsystemd'
        expected.mkdir(0o700)
        env['TEST_DIRECTORY'] = str(expected)
        dirs = basedirs.BaseDirectories(dir_spec, env)
        assert dirs.storage_path() == expected

    def test_storage_path_systemd_mixed_modes(self, dir_spec, env, tmp_path):
        rodir = tmp_path / 'rodir'
        rodir.mkdir(0o500)
        expected = tmp_path / 'rwdir'
        expected.mkdir(0o700)
        env['TEST_DIRECTORY'] = f'{rodir}:{expected}'
        dirs = basedirs.BaseDirectories(dir_spec, env)
        assert dirs.storage_path() == expected

    def test_storage_path_xdg_home(self, dir_spec, env, tmp_path):
        expected = tmp_path / '.xdghome' / 'arvados'
        env['XDG_TEST_HOME'] = str(expected.parent)
        dirs = basedirs.BaseDirectories(dir_spec, env)
        assert dirs.storage_path() == expected
        exp_mode = stat.S_IFDIR | stat.S_IWUSR
        assert (expected.stat().st_mode & exp_mode) == exp_mode

    def test_storage_path_default(self, dir_spec, env, tmp_path):
        expected = tmp_path / dir_spec.xdg_home_default / 'arvados'
        dirs = basedirs.BaseDirectories(dir_spec, env)
        assert dirs.storage_path() == expected
        exp_mode = stat.S_IFDIR | stat.S_IWUSR
        assert (expected.stat().st_mode & exp_mode) == exp_mode

    @pytest.mark.parametrize('subdir,mode', [
        ('str/dir', 0o750),
        (Path('sub', 'path'), 0o770),
    ])
    def test_storage_path_subdir(self, dir_spec, env, umask, tmp_path, subdir, mode):
        expected = tmp_path / dir_spec.xdg_home_default / 'arvados' / subdir
        dirs = basedirs.BaseDirectories(dir_spec, env)
        actual = dirs.storage_path(subdir, mode)
        assert actual == expected
        expect_mode = mode | stat.S_IFDIR
        actual_mode = actual.stat().st_mode
        assert (actual_mode & expect_mode) == expect_mode
        assert not (actual_mode & stat.S_IRWXO)

    def test_empty_xdg_home(self, dir_spec, env, tmp_path):
        env['XDG_TEST_HOME'] = ''
        expected = tmp_path / dir_spec.xdg_home_default / 'emptyhome'
        dirs = basedirs.BaseDirectories(dir_spec, env, expected.name)
        assert dirs.storage_path() == expected

    def test_empty_xdg_dirs(self, dir_spec, env, tmp_path):
        env['XDG_TEST_DIRS'] = ''
        _, _, default_dir = dir_spec.xdg_dirs_default.rpartition(':')
        expected = Path(default_dir, 'empty_dirs')
        expected.parent.mkdir()
        expected.touch()
        dirs = basedirs.BaseDirectories(dir_spec, env, '.')
        actual = list(dirs.search(expected.name))
        assert actual == [expected]

    def test_spec_key_lookup(self):
        dirs = basedirs.BaseDirectories('CACHE')
        assert dirs._spec.systemd_key == 'CACHE_DIRECTORY'
        assert dirs._spec.xdg_dirs_key is None

    def test_spec_enum_lookup(self):
        dirs = basedirs.BaseDirectories(basedirs.BaseDirectorySpecs.CONFIG)
        assert dirs._spec.systemd_key == 'CONFIGURATION_DIRECTORY'
