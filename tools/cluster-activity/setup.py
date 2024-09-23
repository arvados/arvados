#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import os
import sys
import re

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

import arvados_version
version = arvados_version.get_version(SETUP_DIR, "arvados_cluster_activity")

setup(name='arvados-cluster-activity',
      version=version,
      description='Summarize cluster activity from Arvados audit logs and Prometheus metrics',
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license='GNU Affero General Public License, version 3.0',
      packages=['arvados_cluster_activity'],
      include_package_data=True,
      entry_points={"console_scripts": ["arv-cluster-activity=arvados_cluster_activity.main:main"]},
      data_files=[
          ('share/doc/arvados_cluster_activity', ['agpl-3.0.txt']),
      ],
      install_requires=[
          *arvados_version.iter_dependencies(version),
      ],
      extras_require={"prometheus": ["prometheus-api-client"]},
      zip_safe=True,
)
