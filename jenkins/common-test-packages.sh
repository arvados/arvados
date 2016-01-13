#!/bin/sh

set -eu

FAIL=0

echo
shared=$(find -name '*.so')
if test -n "$shared" ; then
    for so in $shared ; do
        if ldd $so | grep "not found" ; then
            echo "^^^ Missing while scanning $so ^^^"
            FAIL=1
        fi
    done
fi

if test -x /jenkins/test-package-$1.sh ; then
    if ! /jenkins/test-package-$1.sh ; then
       FAIL=1
    fi
fi

if test $FAIL = 0 ; then
   echo "Package $1 passed"
fi

exit $FAIL
