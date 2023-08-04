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
version = arvados_version.get_version(SETUP_DIR, "arvados_docker")

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

setup(name="arvados-docker-cleaner",
      version=version,
      description="Arvados Docker cleaner",
      author="Arvados",
      author_email="info@arvados.org",
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license="GNU Affero General Public License version 3.0",
      packages=find_packages(),
      entry_points={
          'console_scripts': ['arvados-docker-cleaner=arvados_docker.cleaner:main'],
      },
      data_files=[
          ('share/doc/arvados-docker-cleaner', ['agpl-3.0.txt', 'arvados-docker-cleaner.service']),
      ],
      install_requires=[
          # The requirements for the docker library broke when requests started
          # supporting urllib3 2.0.
          # See <https://github.com/docker/docker-py/issues/3113>.
          # Make sure we get a version with the bugfix, assuming Python is
          # recent enough.
          'docker>=6.1.0; python_version>"3.6"',
          # If Python is too old, install the latest version we can and pin
          # urllib3 ourselves.
          'docker~=5.0; python_version<"3.7"',
          'urllib3~=1.26; python_version<"3.7"',
          'setuptools',
      ],
      test_suite='tests',
      zip_safe=False
)
