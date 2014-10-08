#!/usr/bin/env python

import os
import subprocess
import time

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or "."
cmd_opts = {'egg_info': {}}
try:
    git_tags = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ct %h', SETUP_DIR]).split()
    assert len(git_tags) == 2
except (AssertionError, OSError, subprocess.CalledProcessError):
    pass
else:
    git_tags[0] = time.strftime('%Y%m%d%H%M%S', time.gmtime(int(git_tags[0])))
    cmd_opts['egg_info']['tag_build'] = '.{}.{}'.format(*git_tags)

setup(name='arvados-node-manager',
      version='0.1',
      description='Arvados compute node manager',
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
      options=cmd_opts,
      )
