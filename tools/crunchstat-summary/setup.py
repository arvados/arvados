#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import os
import sys
import setuptools.command.egg_info as egg_info_cmd

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'

tagger = egg_info_cmd.egg_info
version = os.environ.get("ARVADOS_BUILDING_VERSION")
if not version:
    try:
        import arvados_version
        vtag = arvados_version.VersionInfoFromGit()
        version = vtag.git_latest_tag() + vtag.git_timestamp_tag()
    except ImportError:
        pass


setup(name='crunchstat_summary',
      version=version,
      description='read crunch log files and summarize resource usage',
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
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
          'arvados-python-client',
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
