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
README = os.path.join(arvados_version.SETUP_DIR, 'README.rst')

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
          *arvados_version.iter_dependencies(version),
      ],
      python_requires="~=3.8",
      test_suite='tests',
      zip_safe=False,
)
