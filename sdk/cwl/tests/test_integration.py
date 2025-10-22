# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import subprocess

import pytest

from . import TESTS_DIR, run_cwltest

@pytest.mark.integration
def test_arvados_cwltest(acr_script, integration_colls):
    cwltest = run_cwltest(
        TESTS_DIR / 'arvados-tests.yml',
        acr_script,
    )
    assert cwltest.returncode == os.EX_OK


@pytest.mark.integration
def test_set_properties_17004(arv_session, acr_script, jobs_docker_image, tmp_project):
    inp_path = TESTS_DIR / 'scripts/download_all_data.sh'
    acr_proc = subprocess.run([
        str(acr_script),
        '--project-uuid', tmp_project['uuid'],
        '--submit-runner-image', jobs_docker_image,
        str(TESTS_DIR / '17004-output-props.cwl'),
        '--inp', str(inp_path),
    ])
    assert acr_proc.returncode == os.EX_OK
    contents = arv_session.groups().contents(uuid=tmp_project['uuid']).execute()
    for item in contents['items']:
        if (
                item['kind'] == 'arvados#collection'
                and item['properties'].get('type') == 'output'
                and item['properties'].get('foo') == 'bar'
                and item['properties'].get('baz') == inp_path.name
        ):
            break
    else:
        assert False, "did not find collection with output properties"


@pytest.mark.integration
def test_fix_workflow_18888(acr_script, jobs_docker_image):
    # This is a standalone test because the bug was observed with this
    # command line and was thought to be due to command line handling.
    acr_proc = subprocess.run([
        str(acr_script),
        '--submit-runner-image', jobs_docker_image,
        '18888-download_def.cwl',
        '--scripts', 'scripts/',
    ], cwd=TESTS_DIR)
    assert acr_proc.returncode == os.EX_OK
