#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from setuptools import setup, find_packages

import arvados_version
arv_mod = arvados_version.ARVADOS_PYTHON_MODULES['crunchstat_summary']
version = arv_mod.get_version()
setup(name=arv_mod.package_name,
      version=version,
      cmdclass=arvados_version.CMDCLASS,
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
          *arv_mod.iter_dependencies(version=version),
      ],
      python_requires="~=3.8",
      test_suite='tests',
      zip_safe=False,
)
