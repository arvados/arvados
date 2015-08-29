#!/bin/sh

if test -z "$WORKSPACE" ; then
    echo "Must set WORKSPACE"
    exit 1
fi

FAIL=0

for pkg in ./test-packages-*.sh ; do
    if ! $pkg --run-test ; then
        FAIL=1
        echo "$pkg has install errors"
    fi
done

exit $FAIL
