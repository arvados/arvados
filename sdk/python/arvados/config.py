# config.py - configuration settings and global variables for Arvados clients
#
# Arvados configuration settings are taken from $HOME/.config/arvados.
# Environment variables override settings in the config file.

import os
import re

_settings = None
if os.environ.get('HOME') != None:
    default_config_file = os.environ['HOME'] + '/.config/arvados/settings.conf'
else:
    default_config_file = ''

EMPTY_BLOCK_LOCATOR = 'd41d8cd98f00b204e9800998ecf8427e+0'

def initialize(config_file=default_config_file):
    global _settings
    _settings = {}

    # load the specified config file if available
    try:
        _settings = load(config_file)
    except IOError:
        pass

    # override any settings with environment vars
    for var in os.environ:
        if var.startswith('ARVADOS_'):
            _settings[var] = os.environ[var]

def load(config_file):
    cfg = {}
    with open(config_file, "r") as f:
        for config_line in f:
            if re.match('^\s*$', config_line):
                continue
            if re.match('^\s*#', config_line):
                continue
            var, val = config_line.rstrip().split('=', 2)
            cfg[var] = val
    return cfg

def flag_is_true(key):
    return get(key, '').lower() in set(['1', 't', 'true', 'y', 'yes'])

def get(key, default_val=None):
    return settings().get(key, default_val)

def settings():
    if _settings is None:
        initialize()
    return _settings
