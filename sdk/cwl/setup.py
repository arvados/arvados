#!/usr/bin/env python

import os
import sys
import setuptools.command.egg_info as egg_info_cmd

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

try:
    import gittaggers
    tagger = gittaggers.EggInfoFromGit
except ImportError:
    tagger = egg_info_cmd.egg_info

setup(name='arvados-cwl-runner',
      version='1.0',
      description='Arvados Common Workflow Language runner',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='Apache 2.0',
      packages=find_packages(),
      scripts=[
          'bin/cwl-runner',
          'bin/arvados-cwl-runner'
      ],
      install_requires=[
          'cwltool==1.0.20160519182434',
          'arvados-python-client>=0.1.20160322001610'
      ],
      test_suite='tests',
      tests_require=['mock>=1.0'],
      zip_safe=True,
      cmdclass={'egg_info': tagger},
      )
