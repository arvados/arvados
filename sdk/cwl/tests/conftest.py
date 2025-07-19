# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import http
import os
import shutil
import subprocess
import sys

from pathlib import Path

import arvados
import arvados_cwl._version as acr_version
import pytest

from . import TESTS_DIR
from arvados.collection import Collection
from arvados.commands import keepdocker

def _ensure_collection(arv_client, coll_pdh, upload_root, glob='*'):
    """Ensure a collection exists with the given portable data hash

    This fixture first tries to load the collection from Arvados. If it is not
    found, a new one is created from the files in `upload_root` matching `glob`.
    The collection's API record is returned.
    """
    try:
        coll = Collection(coll_pdh)
    except arvados.errors.ApiError as error:
        assert error.status_code == http.HTTPStatus.NOT_FOUND
        coll = Collection()
        upload_root = TESTS_DIR / upload_root
        for path in upload_root.rglob(glob):
            if path.is_dir():
                continue
            coll_path = str(path.relative_to(upload_root))
            with path.open('rb') as src_file, coll.open(coll_path, 'wb') as dst_file:
                shutil.copyfileobj(src_file, dst_file)
        assert coll.portable_data_hash() == coll_pdh
        coll.save_new()
    return coll.api_response()


@pytest.fixture(scope='session')
def acr_script(tmp_path_factory):
    """Return an executable path to run the tested version of a-c-r"""
    bin_dir = tmp_path_factory.mktemp('bin.')
    bin_path = bin_dir / 'test-arvados-cwl-runner'
    bin_path.touch(0o755)
    with bin_path.open('w') as bin_file:
        bin_file.write(f"""\
#!{sys.executable}
import sys
sys.argv[0] = 'arvados-cwl-runner'
sys.path = {sys.path!r}
import arvados_cwl
sys.exit(arvados_cwl.main())
""")
    return bin_path


@pytest.fixture(scope='session')
def arv_session():
    return arvados.api('v1')


@pytest.fixture(scope='session')
def arv_session_config(arv_session):
    return arv_session.configs().get().execute()


@pytest.fixture(scope='session')
def coll_hellos(arv_session):
    return _ensure_collection(
        arv_session,
        '4d8a70b1e63b2aad6984e40e338e2373+69',
        'secondaryFiles',
        'hello.txt*',
    )


@pytest.fixture(scope='session')
def coll_hg19(arv_session):
    return _ensure_collection(
        arv_session,
        'f225e6259bdd63bc7240599648dde9f1+97',
        'hg19',
    )


@pytest.fixture(scope='session')
def coll_sample1(arv_session):
    return _ensure_collection(
        arv_session,
        '20850f01122e860fb878758ac1320877+71',
        'samples',
        'sample1_S01_R1_001.fastq.gz',
    )


@pytest.fixture(scope='session')
def coll_testdir(arv_session):
    return _ensure_collection(
        arv_session,
        'd7514270f356df848477718d58308cc4+94',
        'testdir',
    )


@pytest.fixture(scope='session')
def jobs_docker_image(arv_session):
    image_name = 'arvados/jobs'
    image_tag = acr_version.__version__
    image_fullname = f'{image_name}:{image_tag}'

    # We must have the image in our local repository. Otherwise we'll try to
    # pull it, which won't work for development images.
    image_listing = subprocess.run(
        ['docker', 'image', 'list', '--quiet', image_fullname],
        capture_output=True,
        text=True,
    )
    assert image_listing.returncode == os.EX_OK, f"Failed to query Docker: {image_listing.stderr}"
    if not image_listing.stdout.strip():
        build_env = os.environ.copy()
        try:
            workspace = Path(os.environ['WORKSPACE'])
        except KeyError:
            workspace = TESTS_DIR.parent.parent.parent
            build_env['WORKSPACE'] = str(workspace)
        build_proc = subprocess.run([
            sys.executable,
            str(workspace / 'build/build_docker_image.py'),
            '--tag', image_fullname,
            image_name,
        ], env=build_env)
        assert build_proc.returncode == os.EX_OK, f"Failed to build {image_name}"

    # Now upload it to our cluster. arv-keepdocker automatically works to avoid
    # redundant uploads, so we're leaning on that here.
    try:
        keepdocker.main(
            [image_name, image_tag],
            install_sig_handlers=False,
            api=arv_session,
        )
    except SystemExit as exit_err:
        assert not exit_err.args[0], f"Failed to uplaod {image_name}"
    return image_fullname


@pytest.fixture(scope='session')
def integration_colls(coll_hellos, coll_hg19, coll_sample1, coll_testdir, jobs_docker_image):
    return [coll_hellos, coll_hg19, coll_sample1, coll_testdir, jobs_docker_image]


@pytest.fixture
def tmp_project(request, arv_session):
    project = arv_session.groups().create(
        body={'group': {
            'name': f'Arvados CWL {request.function.__name__} work',
            'group_class': 'project',
        }},
        ensure_unique_name=True,
    ).execute()
    yield project
    arv_session.groups().delete(uuid=project['uuid']).execute()
