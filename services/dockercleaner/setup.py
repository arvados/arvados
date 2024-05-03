#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import os
import sys
import re

from setuptools import setup, find_packages

import arvados_version
version = arvados_version.get_version()
short_tests_only = arvados_version.short_tests_only()
README = os.path.join(arvados_version.SETUP_DIR, 'README.rst')

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
          *arvados_version.iter_dependencies(version),
          'docker>=6.1.0',
          'setuptools',
      ],
      python_requires="~=3.8",
      test_suite='tests',
      zip_safe=False
)
