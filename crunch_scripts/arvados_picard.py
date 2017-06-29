# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import re
import os
import sys
import fcntl
import subprocess

picard_install_path = None

def install_path():
    global picard_install_path
    if picard_install_path:
        return picard_install_path
    zipball = arvados.current_job()['script_parameters']['picard_zip']
    extracted = arvados.util.zipball_extract(
        zipball = zipball,
        path = 'picard')
    for f in os.listdir(extracted):
        if (re.search(r'^picard-tools-[\d\.]+$', f) and
            os.path.exists(os.path.join(extracted, f, '.'))):
            picard_install_path = os.path.join(extracted, f)
            break
    if not picard_install_path:
        raise Exception("picard-tools-{version} directory not found in %s" %
                        zipball)
    return picard_install_path

def run(module, **kwargs):
    kwargs.setdefault('cwd', arvados.current_task().tmpdir)
    execargs = ['java',
                '-Xmx1500m',
                '-Djava.io.tmpdir=' + arvados.current_task().tmpdir,
                '-jar', os.path.join(install_path(), module + '.jar')]
    execargs += [str(arg) for arg in kwargs.pop('args', [])]
    for key, value in kwargs.pop('params', {}).items():
        execargs += [key.upper() + '=' + str(value)]
    sys.stderr.write("%s.run: exec %s\n" % (__name__, str(execargs)))
    return arvados.util.run_command(execargs, **kwargs)
