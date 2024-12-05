# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This file runs in one of three modes:
#
# 1. If the ARVADOS_BUILDING_VERSION environment variable is set, it writes
#    _version.py and generates dependencies based on that value.
# 2. If running from an arvados Git checkout, it writes _version.py
#    and generates dependencies from Git.
# 3. Otherwise, we expect this is source previously generated from Git, and
#    it reads _version.py and generates dependencies from it.

import os
import re
import runpy
import subprocess
import sys

from pathlib import Path

# These maps explain the relationships between different Python modules in
# the arvados repository. We use these to help generate setup.py.
PACKAGE_DEPENDENCY_MAP = {
    'arvados-cwl-runner': ['arvados-python-client', 'crunchstat_summary'],
    'arvados-user-activity': ['arvados-python-client'],
    'arvados_fuse': ['arvados-python-client'],
    'crunchstat_summary': ['arvados-python-client'],
    'arvados_cluster_activity': ['arvados-python-client'],
}
PACKAGE_MODULE_MAP = {
    'arvados-cwl-runner': 'arvados_cwl',
    'arvados-docker-cleaner': 'arvados_docker',
    'arvados-python-client': 'arvados',
    'arvados-user-activity': 'arvados_user_activity',
    'arvados_fuse': 'arvados_fuse',
    'crunchstat_summary': 'crunchstat_summary',
    'arvados_cluster_activity': 'arvados_cluster_activity',
}
PACKAGE_SRCPATH_MAP = {
    'arvados-cwl-runner': Path('sdk', 'cwl'),
    'arvados-docker-cleaner': Path('services', 'dockercleaner'),
    'arvados-python-client': Path('sdk', 'python'),
    'arvados-user-activity': Path('tools', 'user-activity'),
    'arvados_fuse': Path('services', 'fuse'),
    'crunchstat_summary': Path('tools', 'crunchstat-summary'),
    'arvados_cluster_activity': Path('tools', 'cluster-activity'),
}

ENV_VERSION = os.environ.get("ARVADOS_BUILDING_VERSION")
SETUP_DIR = Path(__file__).absolute().parent
try:
    REPO_PATH = Path(subprocess.check_output(
        ['git', '-C', str(SETUP_DIR), 'rev-parse', '--show-toplevel'],
        stderr=subprocess.DEVNULL,
        text=True,
    ).rstrip('\n'))
except (subprocess.CalledProcessError, OSError):
    REPO_PATH = None
else:
    # Verify this is the arvados monorepo
    if all((REPO_PATH / path).exists() for path in PACKAGE_SRCPATH_MAP.values()):
        PACKAGE_NAME, = (
            pkg_name for pkg_name, path in PACKAGE_SRCPATH_MAP.items()
            if (REPO_PATH / path) == SETUP_DIR
        )
        MODULE_NAME = PACKAGE_MODULE_MAP[PACKAGE_NAME]
        VERSION_SCRIPT_PATH = Path(REPO_PATH, 'build', 'version-at-commit.sh')
    else:
        REPO_PATH = None
if REPO_PATH is None:
    (PACKAGE_NAME, MODULE_NAME), = (
        (pkg_name, mod_name)
        for pkg_name, mod_name in PACKAGE_MODULE_MAP.items()
        if (SETUP_DIR / mod_name).is_dir()
    )

def git_log_output(path, *args):
    return subprocess.check_output(
        ['git', '-C', str(REPO_PATH),
         'log', '--first-parent', '--max-count=1',
         *args, str(path)],
        text=True,
    ).rstrip('\n')

def choose_version_from():
    ver_paths = [SETUP_DIR, VERSION_SCRIPT_PATH, *(
        PACKAGE_SRCPATH_MAP[pkg]
        for pkg in PACKAGE_DEPENDENCY_MAP.get(PACKAGE_NAME, ())
    )]
    getver = max(ver_paths, key=lambda path: git_log_output(path, '--format=format:%ct'))
    print(f"Using {getver} for version number calculation of {SETUP_DIR}", file=sys.stderr)
    return getver

def git_version_at_commit():
    curdir = choose_version_from()
    myhash = git_log_output(curdir, '--format=%H')
    return subprocess.check_output(
        [str(VERSION_SCRIPT_PATH), myhash],
        text=True,
    ).rstrip('\n')

def save_version(setup_dir, module, v):
    with Path(setup_dir, module, '_version.py').open('w') as fp:
        print(f"__version__ = {v!r}", file=fp)

def read_version(setup_dir, module):
    file_vars = runpy.run_path(Path(setup_dir, module, '_version.py'))
    return file_vars['__version__']

def get_version(setup_dir=SETUP_DIR, module=MODULE_NAME):
    if ENV_VERSION:
        version = ENV_VERSION
    elif REPO_PATH is None:
        return read_version(setup_dir, module)
    else:
        version = git_version_at_commit()
    version = version.replace("~dev", ".dev").replace("~rc", "rc").lstrip("development-")
    save_version(setup_dir, module, version)
    return version

def iter_dependencies(version=None):
    if version is None:
        version = get_version()
    # A packaged development release should be installed with other
    # development packages built from the same source, but those
    # dependencies may have earlier "dev" versions (read: less recent
    # Git commit timestamps). This compatible version dependency
    # expresses that as closely as possible. Allowing versions
    # compatible with .dev0 allows any development release.
    # Regular expression borrowed partially from
    # <https://packaging.python.org/en/latest/specifications/version-specifiers/#version-specifiers-regex>
    dep_ver, match_count = re.subn(r'\.dev(0|[1-9][0-9]*)$', '.dev0', version, 1)
    dep_op = '~=' if match_count else '=='
    for dep_pkg in PACKAGE_DEPENDENCY_MAP.get(PACKAGE_NAME, ()):
        yield f'{dep_pkg}{dep_op}{dep_ver}'

# Called from calculate_python_sdk_cwl_package_versions() in run-library.sh
if __name__ == '__main__':
    print(get_version())
