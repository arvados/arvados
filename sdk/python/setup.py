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
        'python-gflags',
        'google-api-python-client',
        'httplib2',
        'requests>=2.4',
        'urllib3',
        'ws4py'
        ],
      test_suite='tests',
      tests_require=['mock>=1.0', 'PyYAML'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
