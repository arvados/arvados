#!/usr/bin/env python3
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
if os.environ.get('ARVADOS_BUILDING_VERSION', False):
    pysdk_dep = "=={}".format(version)
else:
    # On dev releases, arvados-python-client may have a different timestamp
    pysdk_dep = "<={}".format(version)

setup(name='arvados-cwl-runner',
      version=version,
      description='Arvados Common Workflow Language runner',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/arvados/arvados.git",
      license='Apache 2.0',
      packages=find_packages(),
      package_data={'arvados_cwl': ['arv-cwl-schema-v1.0.yml', 'arv-cwl-schema-v1.1.yml', 'arv-cwl-schema-v1.2.yml']},
      scripts=[
          'bin/cwl-runner',
          'bin/arvados-cwl-runner',
      ],
      # Note that arvados/build/run-build-packages.sh looks at this
      # file to determine what version of cwltool and schema-salad to
      # build.
      install_requires=[
          'cwltool==3.0.20210319143721',
          'schema-salad==7.1.20210611090601',
          'arvados-python-client{}'.format(pysdk_dep),
          'setuptools',
          'ciso8601 >= 2.0.0',
          'networkx < 2.6'
      ],
      extras_require={
          ':os.name=="posix" and python_version<"3"': ['subprocess32 >= 3.5.1'],
          ':python_version<"3"': ['pytz'],
      },
      data_files=[
          ('share/doc/arvados-cwl-runner', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      python_requires=">=3.5, <4",
      classifiers=[
          'Programming Language :: Python :: 3',
      ],
      test_suite='tests',
      tests_require=[
          'mock>=1.0,<4',
          'subprocess32>=3.5.1',
      ],
      zip_safe=True,
)
