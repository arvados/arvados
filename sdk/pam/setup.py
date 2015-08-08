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

setup(name='arvados-pam',
      version='0.1',
      description='Arvados PAM module',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url='https://arvados.org',
      download_url='https://github.com/curoverse/arvados.git',
      license='Apache 2.0',
      packages=[
          'arvados_pam',
      ],
      scripts=[
      ],
      install_requires=[
          'arvados-python-client>=0.1.20150801000000',
          'pyyaml',
      ],
      test_suite='tests',
      tests_require=['mock>=1.0', 'PyYAML'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
