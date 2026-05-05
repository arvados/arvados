#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# POSIX shell script to simulate editing.
#
# Syntax:
#     writefile.sh SRCFILE [-a] DSTFILE
# where DSTFILE is the file being "edited".
#
# If we set 'writefile.sh SRCFILE [-a]' as $EDITOR, we would then have a "text
# editor" that writes [or appends] the content of SRCFILE to DSTFILE for any
# DSTFILE.
#
# See the PyTest fixture "setup_editor_simulator" in test_arvcli.py for usage.
SRCFILE="$1"; shift
exec tee "$@" < "${SRCFILE}" > /dev/null
