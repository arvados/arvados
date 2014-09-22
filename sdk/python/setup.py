#!/usr/bin/env python

import os
import subprocess

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__)
README = os.path.join(SETUP_DIR, 'README.rst')

cmd_opts = {'egg_info': {}}
try:
    git_tags = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ci %h', SETUP_DIR]).split()
    assert len(git_tags) == 4
except (AssertionError, OSError, subprocess.CalledProcessError):
    pass
else:
    del git_tags[2]    # Remove timezone
    for ii in [0, 1]:  # Remove non-digits from other datetime fields
        git_tags[ii] = ''.join(c for c in git_tags[ii] if c.isdigit())
    cmd_opts['egg_info']['tag_build'] = '.{}{}.{}'.format(*git_tags)


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
        'bin/arv-get',
        'bin/arv-keepdocker',
        'bin/arv-ls',
        'bin/arv-normalize',
        'bin/arv-put',
        'bin/arv-ws',
        ],
      install_requires=[
        'python-gflags',
        'google-api-python-client',
        'httplib2',
        'urllib3',
        'ws4py'
        ],
      test_suite='tests',
      tests_require=['mock>=1.0', 'PyYAML'],
      zip_safe=False,
      options=cmd_opts,
      )
