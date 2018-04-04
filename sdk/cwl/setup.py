#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import sys
import re

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

env_version = os.environ.get("ARVADOS_BUILDING_VERSION")

def save_version(module, v):
  with open(os.path.join(SETUP_DIR, module, "_version.py"), 'w') as fp:
      return fp.write("__version__ = '%s'\n" % v)

def read_version(module):
  with open(os.path.join(SETUP_DIR, module, "_version.py"), 'r') as fp:
      return re.match("__version__ = '(.*)'$", fp.read()).groups()[0]

if env_version:
    save_version("arvados_cwl", env_version)
else:
    try:
        import arvados_version
        vtag = arvados_version.VersionInfoFromGit()
        save_version("arvados_cwl", vtag.git_latest_tag() + vtag.git_timestamp_tag())
    except ImportError:
        pass

version = read_version("arvados_cwl")

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
          'cwltool==1.0.20180326152342',
          'schema-salad==2.6.20171201034858',
          'typing==3.5.3.0',
          'ruamel.yaml==0.13.7',
          'arvados-python-client>=0.1.20170526013812',
          'setuptools',
          'ciso8601 >=1.0.0, <=1.0.4',
      ],
      data_files=[
          ('share/doc/arvados-cwl-runner', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      test_suite='tests',
      tests_require=['mock>=1.0'],
      zip_safe=True
      )
