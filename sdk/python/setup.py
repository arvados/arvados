#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import sys
import setuptools.command.egg_info as egg_info_cmd

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

tagger = egg_info_cmd.egg_info
version = os.environ.get("ARVADOS_BUILDING_VERSION")
if not version:
    version = "0.1"
    try:
        import gittaggers
        tagger = gittaggers.EggInfoFromGit
    except ImportError:
        pass

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

setup(name='arvados-python-client',
      version=version,
      description='Arvados client library',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='Apache 2.0',
      packages=find_packages(),
      scripts=[
          'bin/arv-copy',
          'bin/arv-get',
          'bin/arv-keepdocker',
          'bin/arv-ls',
          'bin/arv-migrate-docker19',
          'bin/arv-normalize',
          'bin/arv-put',
          'bin/arv-run',
          'bin/arv-ws'
      ],
      data_files=[
          ('share/doc/arvados-python-client', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      install_requires=[
          'ciso8601 >=1.0.0, <=1.0.4',
          'future',
          'google-api-python-client >=1.6.2, <1.7',
          'httplib2 >=0.9.2',
          'pycurl >=7.19.5.1',
          'ruamel.yaml >=0.13.7',
          'setuptools',
          'ws4py <0.4',
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0', 'PyYAML'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
