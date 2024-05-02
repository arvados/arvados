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

setup(name='arvados-user-activity',
      version=version,
      description='Summarize user activity from Arvados audit logs',
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license='GNU Affero General Public License, version 3.0',
      packages=['arvados_user_activity'],
      include_package_data=True,
      entry_points={"console_scripts": ["arv-user-activity=arvados_user_activity.main:main"]},
      data_files=[
          ('share/doc/arvados_user_activity', ['agpl-3.0.txt']),
      ],
      install_requires=[
          *arvados_version.iter_dependencies(version),
      ],
      python_requires="~=3.8",
      zip_safe=True,
)
