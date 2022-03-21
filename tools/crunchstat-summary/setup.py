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
version = arvados_version.get_version(SETUP_DIR, "crunchstat_summary")
if os.environ.get('ARVADOS_BUILDING_VERSION', False):
    pysdk_dep = "=={}".format(version)
else:
    # On dev releases, arvados-python-client may have a different timestamp
    pysdk_dep = "<={}".format(version)

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

setup(name='crunchstat_summary',
      version=version,
      description='Arvados crunchstat-summary reads crunch log files and summarizes resource usage',
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license='GNU Affero General Public License, version 3.0',
      packages=['crunchstat_summary'],
      include_package_data=True,
      scripts=[
          'bin/crunchstat-summary'
      ],
      data_files=[
          ('share/doc/crunchstat_summary', ['agpl-3.0.txt']),
      ],
      install_requires=[
          'arvados-python-client{}'.format(pysdk_dep),
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0'],
      zip_safe=False,
)
