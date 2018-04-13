# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse
import errno
import os
import logging
import signal
from future.utils import listitems, listvalues
import sys

def _pos_int(s):
    num = int(s)
    if num < 0:
        raise ValueError("can't accept negative value: %s" % (num,))
    return num

retry_opt = argparse.ArgumentParser(add_help=False)
retry_opt.add_argument('--retries', type=_pos_int, default=3, help="""
Maximum number of times to retry server requests that encounter temporary
failures (e.g., server down).  Default 3.""")

def _ignore_error(error):
    return None

def _raise_error(error):
    raise error

def make_home_conf_dir(path, mode=None, errors='ignore'):
    # Make the directory path under the user's home directory, making parent
    # directories as needed.
    # If the directory is newly created, and a mode is specified, chmod it
    # with those permissions.
    # If there's an error, return None if errors is 'ignore', else raise an
    # exception.
    error_handler = _ignore_error if (errors == 'ignore') else _raise_error
    tilde_path = os.path.join('~', path)
    abs_path = os.path.expanduser(tilde_path)
    if abs_path == tilde_path:
        return error_handler(ValueError("no home directory available"))
    try:
        os.makedirs(abs_path)
    except OSError as error:
        if error.errno != errno.EEXIST:
            return error_handler(error)
    else:
        if mode is not None:
            os.chmod(abs_path, mode)
    return abs_path

CAUGHT_SIGNALS = [signal.SIGINT, signal.SIGQUIT, signal.SIGTERM]

def exit_signal_handler(sigcode, frame):
    logging.getLogger('arvados').error("Caught signal {}, exiting.".format(sigcode))
    sys.exit(-sigcode)

def install_signal_handlers():
    global orig_signal_handlers
    orig_signal_handlers = {sigcode: signal.signal(sigcode, exit_signal_handler)
                            for sigcode in CAUGHT_SIGNALS}

def restore_signal_handlers():
    for sigcode, orig_handler in listitems(orig_signal_handlers):
        signal.signal(sigcode, orig_handler)
