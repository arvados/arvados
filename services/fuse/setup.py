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
        *arvados_version.iter_dependencies(version),
        'arvados-llfuse >= 1.5.1',
        'python-daemon',
        'ciso8601 >= 2.0.0',
        'setuptools',
        "prometheus_client"
        ],
      python_requires="~=3.8",
      classifiers=[
          'Programming Language :: Python :: 3',
      ],
      test_suite='tests',
      tests_require=['PyYAML', 'parameterized',],
      zip_safe=False
      )
