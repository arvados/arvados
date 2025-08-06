# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import dataclasses
import os
import re
import runpy
import subprocess
import typing as t

from pathlib import Path, PurePath, PurePosixPath

import setuptools
import setuptools.command.build

SETUP_DIR = Path(__file__).absolute().parent
VERSION_SCRIPT_PATH = PurePath('build', 'version-at-commit.sh')

### Metadata generation

@dataclasses.dataclass
class ArvadosPythonPackage:
    package_name: str
    module_name: str
    src_path: PurePath
    dependencies: t.Sequence['ArvadosPythonPackage']

    _VERSION_SUBS = {
        'development-': '',
        '~dev': '.dev',
        '~rc': 'rc',
    }

    def version_file_path(self):
        return PurePath(self.module_name, '_version.py')

    def _workspace_path(self, workdir):
        try:
            workspace = Path(os.environ['WORKSPACE'])
            # This will raise ValueError if they're not related,
            # in which case we don't want to use this $WORKSPACE.
            workdir.relative_to(workspace)
        except (KeyError, ValueError):
            return None
        if (workspace / VERSION_SCRIPT_PATH).exists():
            return workspace
        else:
            return None

    def _git_version(self, workdir):
        workspace = self._workspace_path(workdir)
        if workspace is None:
            return None
        git_log_cmd = [
            'git', 'log', '-n1', '--format=%H', '--',
            str(VERSION_SCRIPT_PATH), str(self.src_path),
        ]
        git_log_cmd.extend(str(dep.src_path) for dep in self.dependencies)
        git_log_proc = subprocess.run(
            git_log_cmd,
            check=True,
            cwd=workspace,
            stdout=subprocess.PIPE,
            text=True,
        )
        version_proc = subprocess.run(
            [str(VERSION_SCRIPT_PATH), git_log_proc.stdout.rstrip('\n')],
            check=True,
            cwd=workspace,
            stdout=subprocess.PIPE,
            text=True,
        )
        return version_proc.stdout.rstrip('\n')

    def _sdist_version(self, workdir):
        try:
            pkg_info = (workdir / 'PKG-INFO').open()
        except FileNotFoundError:
            return None
        with pkg_info:
            for line in pkg_info:
                key, _, val = line.partition(': ')
                if key == 'Version':
                    return val.rstrip('\n')
        raise Exception("found PKG-INFO file but not Version metadata in it")

    def get_version(self, workdir=SETUP_DIR):
        version = (
            # If we're building out of a distribution, we should pass that
            # version through unchanged.
            self._sdist_version(workdir)
            # Otherwise follow the usual Arvados versioning rules.
            or os.environ.get('ARVADOS_BUILDING_VERSION')
            or self._git_version(workdir)
        )
        if not version:
            raise Exception(f"no version information available for {self.package_name}")
        else:
            return re.sub(
                r'(^development-|~dev|~rc)',
                lambda match: self._VERSION_SUBS[match.group(0)],
                version,
            )

    def get_dependencies_version(self, workdir=SETUP_DIR, version=None):
        if version is None:
            version = self.get_version(workdir)
        # A packaged development release should be installed with other
        # development packages built from the same source, but those
        # dependencies may have earlier "dev" versions (read: less recent
        # Git commit timestamps). This compatible version dependency
        # expresses that as closely as possible. Allowing versions
        # compatible with .dev0 allows any development release.
        # Regular expression borrowed partially from
        # <https://packaging.python.org/en/latest/specifications/version-specifiers/#version-specifiers-regex>
        dep_ver, match_count = re.subn(r'\.dev(0|[1-9][0-9]*)$', '.dev0', version, 1)
        return ('~=' if match_count else '==', dep_ver)

    def iter_dependencies(self, workdir=SETUP_DIR, version=None):
        dep_op, dep_ver = self.get_dependencies_version(workdir, version)
        for dep in self.dependencies:
            yield f'{dep.package_name} {dep_op} {dep_ver}'


### Package database

_PYSDK = ArvadosPythonPackage(
    'arvados-python-client',
    'arvados',
    PurePath('sdk', 'python'),
    [],
)
_CRUNCHSTAT_SUMMARY = ArvadosPythonPackage(
    'crunchstat_summary',
    'crunchstat_summary',
    PurePath('tools', 'crunchstat-summary'),
    [_PYSDK],
)
ARVADOS_PYTHON_MODULES = {mod.package_name: mod for mod in [
    _PYSDK,
    _CRUNCHSTAT_SUMMARY,
    ArvadosPythonPackage(
        'arvados-cluster-activity',
        'arvados_cluster_activity',
        PurePath('tools', 'cluster-activity'),
        [_PYSDK],
    ),
    ArvadosPythonPackage(
        'arvados-cwl-runner',
        'arvados_cwl',
        PurePath('sdk', 'cwl'),
        [_PYSDK, _CRUNCHSTAT_SUMMARY],
    ),
    ArvadosPythonPackage(
        'arvados-docker-cleaner',
        'arvados_docker',
        PurePath('services', 'dockercleaner'),
        [],
    ),
    ArvadosPythonPackage(
        'arvados_fuse',
        'arvados_fuse',
        PurePath('services', 'fuse'),
        [_PYSDK],
    ),
    ArvadosPythonPackage(
        'arvados-user-activity',
        'arvados_user_activity',
        PurePath('tools', 'user-activity'),
        [_PYSDK],
    ),
]}

### setuptools integration

class BuildArvadosVersion(setuptools.Command):
    """Write _version.py for an Arvados module"""
    def initialize_options(self):
        self.build_lib = None

    def finalize_options(self):
        self.set_undefined_options("build_py", ("build_lib", "build_lib"))
        arv_mod = ARVADOS_PYTHON_MODULES[self.distribution.get_name()]
        self.out_path = Path(self.build_lib, arv_mod.version_file_path())

    def run(self):
        with self.out_path.open('w') as out_file:
            print(f'__version__ = {self.distribution.get_version()!r}', file=out_file)

    def get_outputs(self):
        return [str(self.out_path)]

    def get_source_files(self):
        return []

    def get_output_mapping(self):
        return {}


class ArvadosBuildCommand(setuptools.command.build.build):
    sub_commands = [
        *setuptools.command.build.build.sub_commands,
        ('build_arvados_version', None),
    ]


CMDCLASS = {
    'build': ArvadosBuildCommand,
    'build_arvados_version': BuildArvadosVersion,
}
