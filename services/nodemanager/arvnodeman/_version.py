import pkg_resources

__version__ = pkg_resources.require('arvados-node-manager')[0].version
