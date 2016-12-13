import pkg_resources

__version__ = pkg_resources.require('arvados_fuse')[0].version
