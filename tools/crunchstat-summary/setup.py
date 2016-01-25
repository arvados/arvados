#!/usr/bin/env python

import os
import sys
import setuptools.command.egg_info as egg_info_cmd

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'

try:
    import gittaggers
    tagger = gittaggers.EggInfoFromGit
except ImportError:
    tagger = egg_info_cmd.egg_info

setup(name='crunchstat_summary',
      version='0.1',
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
