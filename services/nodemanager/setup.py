#!/usr/bin/env python

import os
import sys

from setuptools import setup, find_packages
from setuptools.command.egg_info import egg_info

SETUP_DIR = os.path.dirname(__file__) or "."
README = os.path.join(SETUP_DIR, 'README.rst')

if '--sha1-tag' in sys.argv:
    import gittaggers
    tagger = gittaggers.TagBuildWithCommitDateAndSha1
    sys.argv.remove('--sha1-tag')
else:
    try:
        import gittaggers
        tagger = gittaggers.TagBuildWithCommitDate
    except ImportError:
        tagger = egg_info

setup(name='arvados-node-manager',
      version='0.1',
      description='Arvados compute node manager',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      license='GNU Affero General Public License, version 3.0',
      packages=find_packages(),
      install_requires=[
        'apache-libcloud',
        'arvados-python-client',
        'pykka',
        'python-daemon',
        ],
      scripts=['bin/arvados-node-manager'],
      test_suite='tests',
      tests_require=['mock>=1.0'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
