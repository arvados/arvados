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

setup(name='arvados_fuse',
      version='0.1',
      description='Arvados FUSE driver',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='GNU Affero General Public License, version 3.0',
      packages=['arvados_fuse'],
      scripts=[
        'bin/arv-mount'
        ],
      data_files=[
          ('share/doc/arvados_fuse', ['agpl-3.0.txt', 'README.rst']),
      ],
      install_requires=[
        'arvados-python-client >= 0.1.20151118035730',
        'llfuse >= 1.0',
        'python-daemon',
        'ciso8601',
        'setuptools'
        ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0', 'PyYAML'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
