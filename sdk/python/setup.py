#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import distutils.command.build
import os
import setuptools
import sys
import re

from pathlib import Path
from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

import arvados_version
version = arvados_version.get_version(SETUP_DIR, "arvados")

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

class BuildDiscoveryPydoc(setuptools.Command):
    """Run discovery2pydoc as part of the build process

    This class implements a setuptools subcommand, so it follows
    [the SubCommand protocol][1]. Most of these methods are required by that
    protocol, except `should_run`, which we register as the subcommand
    predicate.

    [1]: https://setuptools.pypa.io/en/latest/userguide/extension.html#setuptools.command.build.SubCommand
    """
    NAME = 'discovery2pydoc'
    description = "build skeleton Python from the Arvados discovery document"
    editable_mode = False
    user_options = [
        ('discovery-json=', 'J', 'JSON discovery document used to build pydoc'),
        ('discovery-output=', 'O', 'relative path to write discovery document pydoc'),
    ]

    def initialize_options(self):
        self.build_lib = None
        self.discovery_json = 'arvados-v1-discovery.json'
        self.discovery_output = str(Path('arvados', 'api_resources.py'))

    def _relative_path(self, src, optname):
        retval = Path(src)
        if retval.is_absolute():
            raise Exception("--{optname} should be a relative path")
        else:
            return retval

    def finalize_options(self):
        # Set self.build_lib to match whatever the build_py subcommand uses.
        self.set_undefined_options('build_py', ('build_lib', 'build_lib'))
        self.json_path = self._relative_path(self.discovery_json, 'discovery-json')
        self.out_path = Path(
            self.build_lib,
            self._relative_path(self.discovery_output, 'discovery-output'),
        )

    def run(self):
        import discovery2pydoc
        self.mkpath(str(self.out_path.parent))
        arglist = ['--output-file', str(self.out_path), str(self.json_path)]
        returncode = discovery2pydoc.main(arglist)
        if returncode != 0:
            raise Exception(f"discovery2pydoc exited {returncode}")

    def should_run(self):
        return True

    def get_outputs(self):
        return [str(self.out_path)]

    def get_source_files(self):
        return [str(self.json_path)]

    def get_output_mapping(self):
        return {
            str(self.json_path): str(self.out_path),
        }
# Run discovery2pydoc as the first subcommand of build.
distutils.command.build.build.sub_commands.insert(
    0, (BuildDiscoveryPydoc.NAME, BuildDiscoveryPydoc.should_run),
)

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
          BuildDiscoveryPydoc.NAME: BuildDiscoveryPydoc,
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
          'setuptools',
          'ws4py >=0.4.2',
          'protobuf<4.0.0dev',
          'pyparsing<3',
          'setuptools>=40.3.0',
      ],
      classifiers=[
          'Programming Language :: Python :: 3',
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0,<4', 'PyYAML', 'parameterized'],
      zip_safe=False
      )
