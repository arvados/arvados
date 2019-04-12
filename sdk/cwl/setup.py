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
          'cwltool==1.0.20181217162649',
          'schema-salad==3.0.20181129082112',
          'typing >= 3.6.4',
          'ruamel.yaml >=0.15.54, <=0.15.77',
          'arvados-python-client>=1.3.0.20190205182514',
          'setuptools',
          'ciso8601 >= 2.0.0',
          'networkx < 2.3'
      ],
      extras_require={
          ':os.name=="posix" and python_version<"3"': ['subprocess32 >= 3.5.1'],
          ':python_version<"3"': ['pytz'],
      },
      data_files=[
          ('share/doc/arvados-cwl-runner', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      classifiers=[
          'Programming Language :: Python :: 2',
          'Programming Language :: Python :: 3',
      ],
      test_suite='tests',
      tests_require=[
          'mock>=1.0',
          'subprocess32>=3.5.1',
      ],
      zip_safe=True
      )
