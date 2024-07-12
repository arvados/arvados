# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# config.py - configuration settings and global variables for Arvados clients
#
# Arvados configuration settings are taken from $HOME/.config/arvados.
# Environment variables override settings in the config file.

import os
import re

from typing import (
    Callable,
    Iterable,
    Union,
)

from . import util
from ._internal import basedirs

_settings = None
default_config_file = ''
"""
.. WARNING:: Deprecated
   Default configuration initialization now searches for the "default"
   configuration in several places. This value no longer has any effect.
"""

KEEP_BLOCK_SIZE = 2**26
EMPTY_BLOCK_LOCATOR = 'd41d8cd98f00b204e9800998ecf8427e+0'

def initialize(
        config_file: Union[
            str,
            os.PathLike,
            Callable[[str], Iterable[os.PathLike]],
        ]=basedirs.BaseDirectories('CONFIG').search,
) -> None:
    global _settings
    _settings = {}

    if callable(config_file):
        search_paths = iter(config_file('settings.conf'))
        config_file = next(search_paths, '')

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
            if re.match(r'^\s*(?:#|$)', config_line):
                continue
            var, val = config_line.rstrip().split('=', 2)
            cfg[var] = val
    return cfg

def flag_is_true(key, d=None):
    if d is None:
        d = settings()
    return d.get(key, '').lower() in set(['1', 't', 'true', 'y', 'yes'])

def get(key, default_val=None):
    return settings().get(key, default_val)

def settings():
    if _settings is None:
        initialize()
    return _settings
