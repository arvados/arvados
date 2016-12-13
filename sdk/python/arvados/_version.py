import pkg_resources

__version__ = pkg_resources.require('arvados-python-client')[0].version
