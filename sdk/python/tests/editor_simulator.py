#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Executable script that simulates a text editor for the purpose of testing
arvcli's "create" and "edit" subcommands.

To use it in the test suites, set the $VISUAL environment variable to the path
of this executable. To get different behaviors, you can include options in the
environment-variable value, e.g.

    VISUAL='editor_simulator.py -i source.json'

This file only provides the executable; to use in a test suite, please refer to
the "setup_editor" fixture in tests/test_arvcli.py for an example.

Usage:

    editor_simulator.py [options] FILE

where FILE is the path to the file being edited. This file must exist.

Options are used to simulate editing behavior, and these include:

    -i/--input INPUT_SOURCE   Put the content of INPUT_SOURCE into FILE. If no
                              INPUT_SOURCE value is specified, leave FILE
                              unmodified.
    -a/--append               Append to FILE, rather than truncating FILE and
                              then injecting the content.
    -r/--replace              Actually move FILE and write new file at the same
                              path (similar to vim with `writebackup`).
    -d/--delete               Delete FILE (will cause `-i` option to be
                              ignored).
    -x/--crash                Open FILE then crash (i.e., exit with code 1).
"""
import argparse
import os
import sys


parser = argparse.ArgumentParser()
parser.add_argument("-i", "--input", dest="input_source", default="")
parser.add_argument("-a", "--append", action="store_true")
parser.add_argument("-r", "--replace", action="store_true")
parser.add_argument("-d", "--delete", action="store_true")
parser.add_argument("-x", "--crash", action="store_true")
parser.add_argument("target_file")

args = parser.parse_args()

if args.crash:
    t = open(args.target_file, "r+")
    os._exit(1)

if args.delete:
    os.unlink(args.target_file)
    sys.exit(0)

if not args.input_source:
    with open(args.target_file, "r+"):
        sys.exit(0)

with open(args.input_source, "r") as f:
    content = f.read()

if args.replace:
    os.rename(args.target_file, f"{args.target_file}.bak")
with open(args.target_file, "r+") as t:
    if args.append:
        t.seek(0, os.SEEK_END)
    else:
        t.truncate()
    t.write(content)
sys.exit(0)
