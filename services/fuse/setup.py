#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import
import os
import sys
import re

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

import arvados_version
version = arvados_version.get_version(SETUP_DIR, "arvados_fuse")
if os.environ.get('ARVADOS_BUILDING_VERSION', False):
    pysdk_dep = "=={}".format(version)
else:
    # On dev releases, arvados-python-client may have a different timestamp
    pysdk_dep = "<={}".format(version)

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

setup(name='arvados_fuse',
      version=version,
      description='Arvados FUSE driver',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license='GNU Affero General Public License, version 3.0',
      packages=['arvados_fuse'],
      scripts=[
        'bin/arv-mount'
        ],
      data_files=[
          ('share/doc/arvados_fuse', ['agpl-3.0.txt', 'README.rst']),
      ],
      install_requires=[
        'arvados-python-client{}'.format(pysdk_dep),
        'llfuse >= 1.3.6',
        'future',
        'python-daemon',
        'ciso8601 >= 2.0.0',
        'setuptools',
        "prometheus_client"
        ],
      extras_require={
          ':python_version<"3"': ['pytz'],
      },
      classifiers=[
          'Programming Language :: Python :: 2',
          'Programming Language :: Python :: 3',
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0', 'PyYAML', 'parameterized',],
      zip_safe=False
      )
