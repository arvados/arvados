#!/usr/bin/env python

import os

from setuptools import setup, find_packages

README = os.path.join(os.path.dirname(__file__), 'README.rst')

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
        'arvados-python-client>=0.1.1411070090',  # 2014-09-18
        'llfuse',
        'python-daemon'
        ],
      test_suite='tests',
      tests_require=['PyYAML'],
      zip_safe=False)
