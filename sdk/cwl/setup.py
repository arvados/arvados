#!/usr/bin/env python

import os
import sys
import subprocess
import setuptools.command.egg_info as egg_info_cmd

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

try:
    import gittaggers
    tagger = gittaggers.EggInfoFromGit
except ImportError:
    tagger = egg_info_cmd.egg_info

versionfile = os.path.join(SETUP_DIR, "arvados_cwl/_version.py")
try:
    gitinfo = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%H', gittaggers.choose_version_from()]).strip()
    with open(versionfile, "w") as f:
        f.write("__version__ = '%s'\n" % gitinfo)
except Exception as e:
    # When installing from package, it won't be part of a git repository, and
    # check_output() will raise an exception.  But the package should include the
    # version file, so we can proceed.
    if not os.path.exists(versionfile):
        raise

setup(name='arvados-cwl-runner',
      version='1.0',
      description='Arvados Common Workflow Language runner',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='Apache 2.0',
      packages=find_packages(),
      package_data={'arvados_cwl': ['arv-cwl-schema.yml']},
      scripts=[
          'bin/cwl-runner',
          'bin/arvados-cwl-runner'
      ],
      # Note that arvados/build/run-build-packages.sh looks at this
      # file to determine what version of cwltool and schema-salad to build.
      install_requires=[
          'cwltool==1.0.20170525215327',
          'schema-salad==2.5.20170428142041',
          'typing==3.5.3.0',
          'ruamel.yaml==0.13.7',
          'arvados-python-client>=0.1.20170526013812',
          'setuptools',
          'ciso8601'
      ],
      data_files=[
          ('share/doc/arvados-cwl-runner', ['LICENSE-2.0.txt', 'README.rst']),
      ],
      test_suite='tests',
      tests_require=['mock>=1.0'],
      zip_safe=True,
      cmdclass={'egg_info': tagger},
      )
