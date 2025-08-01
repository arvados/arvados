#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from setuptools import setup, find_packages

import arvados_version
arv_mod = arvados_version.ARVADOS_PYTHON_MODULES['arvados-docker-cleaner']
version = arv_mod.get_version()
setup(name=arv_mod.package_name,
      version=version,
      cmdclass=arvados_version.CMDCLASS,
      description="Arvados Docker cleaner",
      author="Arvados",
      author_email="info@arvados.org",
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license="GNU Affero General Public License version 3.0",
      packages=find_packages(include=[
          arv_mod.module_name,
          f'{arv_mod.module_name}.*',
      ]),
      entry_points={
          'console_scripts': ['arvados-docker-cleaner=arvados_docker.cleaner:main'],
      },
      data_files=[
          ('share/doc/arvados-docker-cleaner', ['agpl-3.0.txt', 'arvados-docker-cleaner.service']),
      ],
      install_requires=[
          *arv_mod.iter_dependencies(version=version),
          'docker>=6.1.0',
          'setuptools',
      ],
      python_requires="~=3.8",
      test_suite='tests',
      zip_safe=False
)
