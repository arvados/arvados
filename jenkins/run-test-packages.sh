#!/bin/sh

if test -z "$WORKSPACE" ; then
    echo "Must set WORKSPACE"
    exit 1
fi

FAIL=0

ERRORS=""

for pkg in ./test-packages-*.sh ; do
    echo
    echo "== Running $pkg =="
    echo
    if ! $pkg --run-test ; then
        FAIL=1
        ERRORS="$ERRORS\n$pkg has install errors"
    fi
done

echo $ERRORS
exit $FAIL
