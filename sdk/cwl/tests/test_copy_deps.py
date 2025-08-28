# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import subprocess

import pytest

from . import TESTS_DIR, run_cwltest
from arvados.collection import Collection

WORKFLOW_PATH = TESTS_DIR / '19070-copy-deps.cwl'
EXPECTED_WORKFLOW = WORKFLOW_PATH.read_bytes()

@pytest.fixture
def cmd_19070(acr_script, tmp_project, jobs_docker_image):
    return [
        str(acr_script), '--disable-git',
        '--project-uuid', tmp_project['uuid'],
        '--submit-runner-image', jobs_docker_image,
        str(WORKFLOW_PATH),
    ]


def check_core_contents(arv, group, wf_uuid):
    """Assert `group` contains `wf_uuid` and the workflow collection"""
    contents = arv.groups().contents(uuid=group['uuid']).execute()
    matches = [item for item in contents['items'] if item['uuid'] == wf_uuid]
    assert len(matches) == 1
    for item in contents['items']:
        try:
            coll = Collection(item['portable_data_hash'])
        except KeyError:
            continue
        try:
            with coll.open(WORKFLOW_PATH.name, 'rb') as cwl_file:
                workflow_content =  cwl_file.read()
        except FileNotFoundError:
            continue
        if workflow_content == EXPECTED_WORKFLOW:
            break
    else:
        assert False, "workflow collection not found"
    return contents


def check_dep_contents(arv_or_contents, group):
    """Assert `group` contains the `testdir` collection and arvados/jobs image"""
    try:
        items = arv_or_contents['items']
    except TypeError:
        contents = arv_or_contents.groups().contents(uuid=group['uuid']).execute()
        items = contents['items']
    assert any(
        c['kind'] == 'arvados#collection'
        and c['portable_data_hash'] == 'd7514270f356df848477718d58308cc4+94'
        for c in items
    ), f"couldn't find collection depedency in group {group['uuid']}"
    assert any(
        c['kind'] == 'arvados#collection'
        and c['name'].startswith('Docker image arvados jobs')
        for c in items
    ), f"couldn't find jobs image in group {group['uuid']}"


def check_all_contents(arv, group, wf_uuid):
    """Assert `group` contains both the workflow and its dependencies"""
    contents = check_core_contents(arv, group, wf_uuid)
    check_dep_contents(contents, group)


@pytest.mark.integration
def test_create(arv_session, cmd_19070, tmp_project, integration_colls):
    # Create workflow, by default should also copy dependencies
    cmd_19070.insert(1, '--create-workflow')
    acr_proc = subprocess.run(cmd_19070, capture_output=True, text=True)
    assert acr_proc.returncode == os.EX_OK
    check_all_contents(arv_session, tmp_project, acr_proc.stdout.rstrip('\n'))


@pytest.mark.integration
def test_update(arv_session, cmd_19070, tmp_project, integration_colls):
    # Create workflow, but with --no-copy-deps it shouldn't copy anything
    acr = cmd_19070.pop(0)
    create_cmd = [acr, '--create-workflow', '--no-copy-deps'] + cmd_19070
    create_proc = subprocess.run(create_cmd, capture_output=True, text=True)
    assert create_proc.returncode == os.EX_OK
    wf_uuid = create_proc.stdout.rstrip('\n')
    contents = check_core_contents(arv_session, tmp_project, wf_uuid)
    with pytest.raises(AssertionError):
        check_dep_contents(contents, tmp_project)

    update_cmd = [acr, '--update-workflow', wf_uuid] + cmd_19070
    update_proc = subprocess.run(update_cmd, capture_output=True, text=True)
    assert update_proc.returncode == os.EX_OK
    check_all_contents(arv_session, tmp_project, wf_uuid)


@pytest.mark.integration
def test_execute_without_deps(arv_session, cmd_19070, tmp_project, integration_colls):
    run_proc = subprocess.run(cmd_19070)
    assert run_proc.returncode == os.EX_OK
    contents = arv_session.groups().contents(uuid=tmp_project['uuid']).execute()
    # container request+log+container log+step output+final output == 5 items
    assert len(contents['items']) == 5
    assert not any(item['kind'] == 'arvados#workflow' for item in contents['items'])


@pytest.mark.integration
def test_execute_with_deps(arv_session, cmd_19070, tmp_project, integration_colls):
    cmd_19070.insert(1, '--copy-deps')
    run_proc = subprocess.run(cmd_19070)
    assert run_proc.returncode == os.EX_OK
    check_dep_contents(arv_session, tmp_project)
