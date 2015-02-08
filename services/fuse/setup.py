#!/usr/bin/env python

import os
import sys

from setuptools import setup, find_packages
from setuptools.command.egg_info import egg_info

SETUP_DIR = os.path.dirname(__file__) or '.'
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

setup(name='arvados_fuse',
      version='0.1',
      description='Arvados FUSE driver',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='GNU Affero General Public License, version 3.0',
      packages=find_packages(),
      scripts=[
        'bin/arv-mount'
        ],
      install_requires=[
        'arvados-python-client>=0.1.20141203150737.277b3c7',
        'llfuse',
        'python-daemon',
        ],
      test_suite='tests',
      tests_require=['PyYAML'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
