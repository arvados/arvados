from setuptools import setup
import setup_header

setup(name='arvados-fuse-driver',
      version='0.1.' + setup_header.minor_version,
      description='Arvados FUSE driver',
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='Apache 2.0',
      packages=['arvados.fuse'],
      scripts=[
        'bin/arv-mount'
        ],
      install_requires=[
        'arvados-python-client',
        'llfuse',
        'python-daemon'
        ],
      zip_safe=False)
