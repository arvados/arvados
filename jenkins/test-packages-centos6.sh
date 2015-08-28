#!/bin/sh

if test "$1" = --run-test ; then

    if test -z "$WORKSPACE" ; then
        echo "Must set WORKSPACE"
        exit 1
    fi

    exec docker run \
         --rm \
         --volume=$WORKSPACE/packages/centos6:/mnt \
         --volume=$(readlink -f $0):/root/run-test.sh \
         --workdir=/mnt \
         centos:6 \
         /root/run-test.sh --install-scl
fi

if test "$1" = --install-scl ; then
    yum install --assumeyes scl-utils
    curl -L -O https://www.softwarecollections.org/en/scls/rhscl/python27/epel-6-x86_64/download/rhscl-python27-epel-6-x86_64.noarch.rpm
    yum install --assumeyes rhscl-python27-epel-6-x86_64.noarch.rpm
    yum install --assumeyes python27
    exec scl enable python27 $0
fi

yum install --assumeyes python27-python*.rpm

mkdir -p /tmp/opts
cd /tmp/opts

for r in /mnt/python27-python-*x86_64.rpm ; do
    rpm2cpio $r | cpio -idm
done

FAIL=0

for so in $(find . -name "*.so") ; do
    if ldd $so | grep "not found" ; then
        echo "^^^ Missing while scanning $so ^^^"
        FAIL=1
    fi
done

python <<EOF
import arvados
import arvados_fuse
EOF

exit $FAIL
