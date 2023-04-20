#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import os
import sys
import re

from pathlib import Path
from setuptools import setup, find_packages
from setuptools.command import build_py

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

import arvados_version
version = arvados_version.get_version(SETUP_DIR, "arvados")

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

class BuildPython(build_py.build_py):
    """Extend setuptools `build_py` to generate API documentation

    This class implements a setuptools subcommand, so it follows
    [the SubCommand protocol][1]. Most of these methods are required by that
    protocol, except `should_run`, which we register as the subcommand
    predicate.

    [1]: https://setuptools.pypa.io/en/latest/userguide/extension.html#setuptools.command.build.SubCommand
    """
    # This is implemented as functionality on top of `build_py`, rather than a
    # dedicated subcommand, because that's the only way I can find to run this
    # code during both `build` and `install`. setuptools' `install` command
    # normally calls specific `build` subcommands directly, rather than calling
    # the entire command, so it skips custom subcommands.
    user_options = build_py.build_py.user_options + [
        ('discovery-json=', 'J', 'JSON discovery document used to build pydoc'),
        ('discovery-output=', 'O', 'relative path to write discovery document pydoc'),
    ]

    def initialize_options(self):
        super().initialize_options()
        self.discovery_json = 'arvados-v1-discovery.json'
        self.discovery_output = str(Path('arvados', 'api_resources.py'))

    def _relative_path(self, src, optname):
        retval = Path(src)
        if retval.is_absolute():
            raise Exception(f"--{optname} should be a relative path")
        else:
            return retval

    def finalize_options(self):
        super().finalize_options()
        self.json_path = self._relative_path(self.discovery_json, 'discovery-json')
        self.out_path = Path(
            self.build_lib,
            self._relative_path(self.discovery_output, 'discovery-output'),
        )

    def run(self):
        super().run()
        import discovery2pydoc
        arglist = ['--output-file', str(self.out_path), str(self.json_path)]
        returncode = discovery2pydoc.main(arglist)
        if returncode != 0:
            raise Exception(f"discovery2pydoc exited {returncode}")

    def get_outputs(self):
        retval = super().get_outputs()
        retval.append(str(self.out_path))
        return retval

    def get_source_files(self):
        retval = super().get_source_files()
        retval.append(str(self.json_path))
        return retval

    def get_output_mapping(self):
        retval = super().get_output_mapping()
        retval[str(self.json_path)] = str(self.out_path)
        return retval


setup(name='arvados-python-client',
      version=version,
      description='Arvados client library',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license='Apache 2.0',
      cmdclass={
          'build_py': BuildPython,
      },
      packages=find_packages(),
      scripts=[
          'bin/arv-copy',
          'bin/arv-get',
          'bin/arv-keepdocker',
          'bin/arv-ls',
          'bin/arv-migrate-docker19',
          'bin/arv-federation-migrate',
          'bin/arv-normalize',
          'bin/arv-put',
          'bin/arv-ws'
      ],
      data_files=[
          ('share/doc/arvados-python-client', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      install_requires=[
          'ciso8601 >=2.0.0',
          'future',
          'google-api-core <2.11.0', # 2.11.0rc1 is incompatible with google-auth<2
          'google-api-python-client >=2.1.0',
          'google-auth<2',
          'httplib2 >=0.9.2, <0.20.2',
          'pycurl >=7.19.5.1, <7.45.0',
          'ruamel.yaml >=0.15.54, <0.17.22',
          'setuptools>=40.3.0',
          'typing_extensions; python_version<"3.8"',
          'ws4py >=0.4.2',
          'protobuf<4.0.0dev',
          'pyparsing<3',
      ],
      classifiers=[
          'Programming Language :: Python :: 3',
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0,<4', 'PyYAML', 'parameterized'],
      zip_safe=False
      )
