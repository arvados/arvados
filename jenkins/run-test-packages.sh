#!/bin/sh

if test -z "$WORKSPACE" ; then
    echo "Must set WORKSPACE"
    exit 1
fi

for pkg in test-packages-*.sh ; do
    $pkg --run-test
done
