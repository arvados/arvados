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
      cmdclass={'egg_info': TagBuildWithCommit},
      )
