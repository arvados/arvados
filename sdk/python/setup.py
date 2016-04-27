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

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

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
      data_files=[
          ('share/doc/arvados-python-client', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      install_requires=[
          'google-api-python-client==1.4.2',
          'oauth2client >=1.4.6, <2',
          'pyasn1-modules==0.0.5',
          'ciso8601',
          'httplib2',
          'pycurl >=7.19.5.1, <7.21.5',
          'python-gflags<3.0',
          'ws4py'
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0', 'PyYAML'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
