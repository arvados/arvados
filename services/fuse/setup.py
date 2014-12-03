#!/usr/bin/env python

import os
import subprocess
import time

from setuptools import setup, find_packages
from setuptools.command.egg_info import egg_info

SETUP_DIR = os.path.dirname(__file__)
README = os.path.join(SETUP_DIR, 'README.rst')

class TagBuildWithCommit(egg_info):
    """Tag the build with the sha1 and date of the last git commit.

    If a build tag has already been set (e.g., "egg_info -b", building
    from source package), leave it alone.
    """
    def tags(self):
        if self.tag_build is None:
            git_tags = subprocess.check_output(
                ['git', 'log', '--first-parent', '--max-count=1',
                 '--format=format:%ct %h', SETUP_DIR]).split()
            assert len(git_tags) == 2
            git_tags[0] = time.strftime(
                '%Y%m%d%H%M%S', time.gmtime(int(git_tags[0])))
            self.tag_build = '.{}.{}'.format(*git_tags)
        return egg_info.tags(self)


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
        'arvados-python-client>=0.1.20141203150737.277b3c7',
        'llfuse',
        'python-daemon'
        ],
      test_suite='tests',
      tests_require=['PyYAML'],
      zip_safe=False,
      cmdclass={'egg_info': TagBuildWithCommit},
      )
