# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import re
import os
import sys

rtg_install_path = None

def setup():
    global rtg_install_path
    if rtg_install_path:
        return rtg_install_path
    rtg_path = arvados.util.zipball_extract(
        zipball = arvados.current_job()['script_parameters']['rtg_binary_zip'],
        path = 'rtg')
    rtg_license_path = arvados.util.collection_extract(
        collection = arvados.current_job()['script_parameters']['rtg_license'],
        path = 'license',
        decompress = False)

    # symlink to rtg-license.txt
    license_txt_path = os.path.join(rtg_license_path, 'rtg-license.txt')
    try:
        os.symlink(license_txt_path, os.path.join(rtg_path,'rtg-license.txt'))
    except OSError:
        if not os.path.exists(os.path.join(rtg_path,'rtg-license.txt')):
            os.symlink(license_txt_path, os.path.join(rtg_path,'rtg-license.txt'))

    rtg_install_path = rtg_path
    return rtg_path

def run_rtg(command, output_dir, command_args, **kwargs):
    global rtg_install_path
    execargs = [os.path.join(rtg_install_path, 'rtg'),
                command,
                '-o', output_dir]
    execargs += command_args
    sys.stderr.write("run_rtg: exec %s\n" % str(execargs))
    arvados.util.run_command(
        execargs,
        cwd=arvados.current_task().tmpdir,
        stderr=sys.stderr,
        stdout=sys.stderr)

    # Exit status cannot be trusted in rtg 1.1.1.
    assert_done(output_dir)

    # Copy log files to stderr and delete them to avoid storing them
    # in Keep with the output data.
    for dirent in arvados.util.listdir_recursive(output_dir):
        if is_log_file(dirent):
            log_file = os.path.join(output_dir, dirent)
            sys.stderr.write(' '.join(['==>', dirent, '<==\n']))
            with open(log_file, 'rb') as f:
                while True:
                    buf = f.read(2**20)
                    if len(buf) == 0:
                        break
                    sys.stderr.write(buf)
            sys.stderr.write('\n') # in case log does not end in newline
            os.unlink(log_file)

def assert_done(output_dir):
    # Sanity-check exit code.
    done_file = os.path.join(output_dir, 'done')
    if not os.path.exists(done_file):
        raise Exception("rtg exited 0 but %s does not exist. abort.\n" % done_file)

def is_log_file(filename):
    return re.search(r'^(.*/)?(progress|done|\S+.log)$', filename)

setup()
