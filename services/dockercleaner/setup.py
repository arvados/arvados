#!/usr/bin/env python3

import os
import sys
import setuptools.command.egg_info as egg_info_cmd

from setuptools import setup, find_packages

try:
    import gittaggers
    tagger = gittaggers.EggInfoFromGit
except ImportError:
    tagger = egg_info_cmd.egg_info

setup(name="arvados-docker-cleaner",
      version="0.1",
      description="Arvados Docker cleaner",
      author="Arvados",
      author_email="info@arvados.org",
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license="GNU Affero General Public License version 3.0",
      packages=find_packages(),
      install_requires=[
        'docker-py',
        ],
      tests_require=[
        'mock',
        ],
      test_suite='tests',
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
