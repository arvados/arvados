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

setup(name='arvados-python-client',
      version='0.1',
      description='Arvados client library',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='Apache 2.0',
      packages=find_packages(),
      scripts=[
          'bin/arv-copy',
          'bin/arv-get',
          'bin/arv-keepdocker',
          'bin/arv-ls',
          'bin/arv-normalize',
          'bin/arv-put',
          'bin/arv-run',
          'bin/arv-ws'
      ],
      install_requires=[
          'google-api-python-client',
          'httplib2',
          'pycurl>=7.19',
          'python-gflags',
          'requests>=2.4',
          'urllib3',
          'ws4py'
      ],
      test_suite='tests',
      tests_require=['mock>=1.0', 'PyYAML'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
