#!/bin/bash

yum -q clean all
touch /var/lib/rpm/*
if ! yum -q install --assumeyes \
     python27-python-arvados-python-client python27-python-arvados-fuse arvados-node-manager
then
    exit 1
fi

mkdir -p /tmp/opts
cd /tmp/opts

for r in /arvados/packages/centos6/python27-python-*x86_64.rpm ; do
    rpm2cpio $r | cpio -idm
done

for so in $(find -name "*.so") ; do
    echo
    echo "== Packages dependencies for $so =="
    scl enable python27 "ldd $so" \
        | awk '($3 ~ /^\//){print $3}' | sort -u | xargs rpm -qf | sort -u
done

exec scl enable python27 /jenkins/common-test-packages.sh
