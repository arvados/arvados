#!/usr/bin/env python

from setuptools import setup

setup(name='arvados_fuse',
      version='0.1',
      description='Arvados FUSE driver',
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
        'arvados-python-client',
        'llfuse',
        'python-daemon'
        ],
      zip_safe=False)
