#!/bin/bash

set -eu

yum -q clean all
touch /var/lib/rpm/*

export ARV_PACKAGES_DIR=/arvados/packages/centos6

rpm -qa | sort > "$ARV_PACKAGES_DIR/$1.before"

yum install --assumeyes $1

rpm -qa | sort > "$ARV_PACKAGES_DIR/$1.after"

set +e
diff "$ARV_PACKAGES_DIR/$1.before" "$ARV_PACKAGES_DIR/$1.after" >"$ARV_PACKAGES_DIR/$1.diff"
set -e

SCL=""
if scl enable python27 true 2>/dev/null ; then
    SCL="scl enable python27"
fi

mkdir -p /tmp/opts
cd /tmp/opts

rpm2cpio "$ARV_PACKAGES_DIR/$1"-*.rpm | cpio -idm 2>/dev/null

shared=$(find -name '*.so')
if test -n "$shared" ; then
    for so in $shared ; do
        echo
        echo "== Packages dependencies for $so =="
        $SCL ldd "$so" \
            | awk '($3 ~ /^\//){print $3}' | sort -u | xargs rpm -qf | sort -u
    done
fi

if test -n "$SCL" ; then
    exec $SCL "/jenkins/package-testing/common-test-packages.sh '$1'"
else
    exec /jenkins/package-testing/common-test-packages.sh "$1"
fi
