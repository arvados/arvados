#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from setuptools import setup, find_packages

import arvados_version
arv_mod = arvados_version.ARVADOS_PYTHON_MODULES['arvados_fuse']
version = arv_mod.get_version()
setup(name=arv_mod.package_name,
      version=version,
      cmdclass=arvados_version.CMDCLASS,
      description='Arvados FUSE driver',
      long_description=(arvados_version.SETUP_DIR / 'README.rst').read_text(),
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
        *arv_mod.iter_dependencies(version=version),
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
