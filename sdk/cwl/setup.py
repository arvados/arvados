#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import os
import sys

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

import arvados_version
version = arvados_version.get_version(SETUP_DIR, "arvados_cwl")

setup(name='arvados-cwl-runner',
      version=version,
      description='Arvados Common Workflow Language runner',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='Apache 2.0',
      packages=find_packages(),
      package_data={'arvados_cwl': ['arv-cwl-schema.yml']},
      scripts=[
          'bin/cwl-runner',
          'bin/arvados-cwl-runner',
      ],
      # Note that arvados/build/run-build-packages.sh looks at this
      # file to determine what version of cwltool and schema-salad to build.
      install_requires=[
          'cwltool==1.0.20180806194258',
          'schema-salad==2.7.20180719125426',
          'typing >= 3.6.4',
          # Need to limit ruamel.yaml version to 0.15.26 because of bug
          # https://bitbucket.org/ruamel/yaml/issues/227/regression-parsing-flow-mapping
          'ruamel.yaml >=0.13.11, <= 0.15.26',
          'arvados-python-client>=1.1.4.20180607143841',
          'setuptools',
          'ciso8601 >=1.0.6, <2.0.0',
          'subprocess32>=3.5.1',
      ],
      data_files=[
          ('share/doc/arvados-cwl-runner', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      test_suite='tests',
      tests_require=[
          'mock>=1.0',
          'subprocess32>=3.5.1',
      ],
      zip_safe=True
      )
