# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import subprocess
import urllib.parse

from pathlib import PurePosixPath

import pytest

from . import run_cwltest

def _ensure_git_clone(git_dir, url, ref):
    """Clone `url` to `git_dir`, check out `ref`, and return `git_dir`"""
    subprocess.run(
        ['git', 'clone', '--quiet', '--no-checkout', url, str(git_dir)],
        check=True,
    )
    subprocess.run(
        ['git', 'switch', '--quiet', '--detach', ref],
        cwd=git_dir,
        check=True,
    )
    yield git_dir


def _ensure_git_worktree(git_dir, work_ref):
    """Create a temporary worktree at `git_dir` from `work_ref`"""
    subprocess.run(
        ['git', 'worktree', 'add', '--quiet', str(git_dir), work_ref],
        check=True,
    )
    yield git_dir
    subprocess.run(
        ['git', 'worktree', 'remove', '--force', str(git_dir)],
        check=True,
    )


def _ensure_git(tmp_path_factory, url, ref, remote_name=None):
    """Create a temporary Git checkout

    If the Git ref `remotes/REMOTE_NAME/ci-build` exists, create a worktree from
    it and yield that. This is provided by the Arvados CI server and fast.
    Otherwise, clone `url`, check out `ref`, and return that directory.
    """
    if remote_name is None:
        url_path = urllib.parse.urlparse(url).path
        assert url_path
        remote_name = PurePosixPath(url_path).stem
    git_dir = tmp_path_factory.mktemp(f'{remote_name}_{ref.replace("/", "_")}_')
    rev_parse = subprocess.run(
        ['git', 'rev-parse', '--verify', f'remotes/{remote_name}/ci-build'],
        capture_output=True,
        text=True,
    )
    if rev_parse.returncode == os.EX_OK:
        yield from _ensure_git_worktree(git_dir, rev_parse.stdout.rstrip('\n'))
    else:
        yield from _ensure_git_clone(git_dir, url, ref)


@pytest.fixture
def badges_dir(request, tmp_path):
    return tmp_path / 'badges'


@pytest.fixture(scope='session')
def cwl1_0git(tmp_path_factory):
    yield from _ensure_git(
        tmp_path_factory,
        'https://github.com/common-workflow-language/common-workflow-language.git',
        'tags/v1.0.2',
        'cwl-v1.0',
    )


@pytest.fixture(scope='session')
def cwl1_1git(tmp_path_factory):
    yield from _ensure_git(
        tmp_path_factory,
        'https://github.com/common-workflow-language/cwl-v1.1.git',
        '3e90671b25f7840ef2926ad2bacbf447772dda94',
    )


@pytest.fixture(scope='session')
def cwl1_2git(tmp_path_factory):
    yield from _ensure_git(
        tmp_path_factory,
        'https://github.com/common-workflow-language/cwl-v1.2.git',
        'tags/v1.2.1',
    )


@pytest.fixture
def skipped_tests_for_config(arv_session_config):
    """Return an appropriate `cwltest -S` option for this Arvados configuration"""
    testnames = []
    try:
        runtime_engine = arv_session_config['Containers']['RuntimeEngine']
    except (KeyError, TypeError):
        runtime_engine = None
    if runtime_engine != 'docker':
        testnames.append('docker_entrypoint')
    if testnames:
        return ['-S', ','.join(testnames)]
    else:
        return []


@pytest.mark.cwl_conformance
@pytest.mark.integration
def test_conformance_1_0(acr_script, badges_dir, cwl1_0git, jobs_docker_image, skipped_tests_for_config):
    cwltest = run_cwltest(
        cwl1_0git / 'v1.0/conformance_test_v1.0.yaml',
        acr_script,
        badges_dir,
        test_args=skipped_tests_for_config,
    )
    assert cwltest.returncode == os.EX_OK


@pytest.mark.cwl_conformance
@pytest.mark.integration
def test_conformance_1_1(acr_script, badges_dir, cwl1_1git, jobs_docker_image, skipped_tests_for_config):
    cwltest = run_cwltest(
        cwl1_1git / 'conformance_tests.yaml',
        acr_script,
        badges_dir,
        test_args=skipped_tests_for_config + ['-N', '199'],
    )
    assert cwltest.returncode == os.EX_OK


@pytest.mark.cwl_conformance
@pytest.mark.integration
def test_conformance_1_2(acr_script, badges_dir, cwl1_2git, jobs_docker_image, skipped_tests_for_config):
    cwltest = run_cwltest(
        cwl1_2git / 'conformance_tests.yaml',
        acr_script,
        badges_dir,
        test_args=skipped_tests_for_config,
    )
    assert cwltest.returncode == os.EX_OK
