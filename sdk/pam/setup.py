#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import glob
import os
import sys
import re
import subprocess

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

import arvados_version
version = arvados_version.get_version(SETUP_DIR, "arvados_pam")
if os.environ.get('ARVADOS_BUILDING_VERSION', False):
    pysdk_dep = "=={}".format(version)
else:
    # On dev releases, arvados-python-client may have a different timestamp
    pysdk_dep = "<={}".format(version)

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

setup(name='arvados-pam',
      version=version,
      description='Arvados PAM module',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url='https://arvados.org',
      download_url='https://github.com/arvados/arvados.git',
      license='Apache 2.0',
      packages=[
          'arvados_pam',
      ],
      scripts=[
      ],
      data_files=[
          ('lib/security', ['lib/libpam_arvados.py']),
          ('share/pam-configs', ['pam-configs/arvados']),
          ('share/doc/arvados-pam', ['LICENSE-2.0.txt', 'README.rst']),
          ('share/doc/arvados-pam/examples', glob.glob('examples/*')),
      ],
      install_requires=[
          'arvados-python-client{}'.format(pysdk_dep),
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0', 'python-pam'],
      zip_safe=False,
)
