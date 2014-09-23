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
        'arvados-python-client>=0.1.1411069908.8ba7f94',  # 2014-09-18
        'llfuse',
        'python-daemon'
        ],
      test_suite='tests',
      tests_require=['PyYAML'],
      zip_safe=False,
      options=cmd_opts,
      )
