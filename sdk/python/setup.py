#!/usr/bin/env python

import os

from setuptools import setup, find_packages

README = os.path.join(os.path.dirname(__file__), 'README.rst')

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
      tests_require=['mock', 'PyYAML'],
      zip_safe=False)
