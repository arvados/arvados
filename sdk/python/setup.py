#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import os
import sys
import re

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

import arvados_version
version = arvados_version.get_version(SETUP_DIR, "arvados")

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
      download_url="https://github.com/arvados/arvados.git",
      license='Apache 2.0',
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
          'google-api-python-client >=1.6.2, <1.7',
          'httplib2 >=0.9.2',
          'pycurl >=7.19.5.1',
          'ruamel.yaml >=0.15.54, <=0.16.5',
          'setuptools',
          'ws4py >=0.4.2',
          'rsa < 4.1'
      ],
      extras_require={
          ':os.name=="posix" and python_version<"3"': ['subprocess32 >= 3.5.1'],
          ':python_version<"3"': ['pytz'],
      },
      classifiers=[
          'Programming Language :: Python :: 2',
          'Programming Language :: Python :: 3',
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0,<4', 'PyYAML'],
      zip_safe=False
      )
