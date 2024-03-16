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
short_tests_only = arvados_version.short_tests_only()

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
          'docker>=6.1.0',
          'setuptools',
      ],
      python_requires="~=3.8",
      test_suite='tests',
      zip_safe=False
)
