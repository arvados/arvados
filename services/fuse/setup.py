#!/usr/bin/env python

import os
import subprocess
import time

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__)
README = os.path.join(SETUP_DIR, 'README.rst')

cmd_opts = {'egg_info': {}}
try:
    git_tags = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ct %h', SETUP_DIR],
        stderr=open('/dev/null','w')).split()
    assert len(git_tags) == 2
except (AssertionError, OSError, subprocess.CalledProcessError):
    pass
else:
    git_tags[0] = time.strftime('%Y%m%d%H%M%S', time.gmtime(int(git_tags[0])))
    cmd_opts['egg_info']['tag_build'] = '.{}.{}'.format(*git_tags)


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
      install_requires=[
        'arvados-python-client>=0.1.20141103223015.68dae83',
        'llfuse',
        'python-daemon'
        ],
      test_suite='tests',
      tests_require=['PyYAML'],
      zip_safe=False,
      options=cmd_opts,
      )
