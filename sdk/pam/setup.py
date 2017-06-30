#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import glob
import os
import sys
import setuptools.command.egg_info as egg_info_cmd
import subprocess

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

tagger = egg_info_cmd.egg_info
try:
    import gittaggers
    tagger = gittaggers.EggInfoFromGit
except (ImportError, OSError):
    pass

setup(name='arvados-pam',
      version='0.1',
      description='Arvados PAM module',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url='https://arvados.org',
      download_url='https://github.com/curoverse/arvados.git',
      license='Apache 2.0',
      packages=[
          'arvados_pam',
      ],
      scripts=[
      ],
      data_files=[
          ('lib/security', ['lib/libpam_arvados.py']),
          ('share/pam-configs', ['pam-configs/arvados']),
          ('share/doc/arvados-pam', ['LICENSE-2.0.txt', 'README.rst']),
          ('share/doc/arvados-pam/examples', glob.glob('examples/*')),

          # The arvados build scripts used to install data files to
          # "/usr/data/*" but now install them to "/usr/*". Here, we
          # install an extra copy in the old location so existing pam
          # configs can still work. When old systems have had a chance
          # to update to the new paths, this line can be removed.
          ('data/lib/security', ['lib/libpam_arvados.py']),
      ],
      install_requires=[
          'arvados-python-client>=0.1.20150801000000',
      ],
      test_suite='tests',
      tests_require=['pbr<1.7.0', 'mock>=1.0', 'python-pam'],
      zip_safe=False,
      cmdclass={'egg_info': tagger},
      )
