#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from setuptools import setup, find_packages

import arvados_version
arv_mod = arvados_version.ARVADOS_PYTHON_MODULES['arvados-cwl-runner']
version = arv_mod.get_version()
setup(name=arv_mod.package_name,
      version=version,
      description='Arvados Common Workflow Language runner',
      long_description=(arvados_version.SETUP_DIR / 'README.rst').read_text(),
      cmdclass=arvados_version.CMDCLASS,
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license='Apache 2.0',
      packages=find_packages(include=[
          arv_mod.module_name,
          f'{arv_mod.module_name}.*',
      ]),
      package_data={'arvados_cwl': ['arv-cwl-schema-v1.0.yml', 'arv-cwl-schema-v1.1.yml', 'arv-cwl-schema-v1.2.yml']},
      entry_points={"console_scripts": ["cwl-runner=arvados_cwl:main",
                                        "arvados-cwl-runner=arvados_cwl:main"],
                    "cwltest.fsaccess": ["fsaccess=arvados_cwl.fsaccess:get_fsaccess"]},
      install_requires=[
          *arv_mod.iter_dependencies(version=version),
          'cwltool==3.1.20240508115724',
          'schema-salad==8.5.20240503091721',
          'ciso8601 >= 2.0.0',
          'setuptools>=40.3.0',
      ],
      data_files=[
          ('share/doc/arvados-cwl-runner', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      python_requires="~=3.8",
      classifiers=[
          'Programming Language :: Python :: 3',
      ],
      test_requires=[
        'parameterized'
      ],
      test_suite='tests',
      zip_safe=True,
)
