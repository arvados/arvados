#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -eu

FAIL=0

echo

while read so && [ -n "$so" ]; do
    if ldd "$so" | grep "not found" ; then
        echo "^^^ Missing while scanning $so ^^^"
        FAIL=1
    fi
done <<EOF
$(find -name '*.so')
EOF

if test -x "/jenkins/package-testing/test-package-$1.sh" ; then
    if ! "/jenkins/package-testing/test-package-$1.sh" ; then
       FAIL=1
    fi
fi

if test $FAIL = 0 ; then
   echo "Package $1 passed"
fi

exit $FAIL
