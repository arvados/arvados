#!/bin/sh

if test "$1" = --run-test ; then

    if test -z "$WORKSPACE" ; then
        echo "Must set WORKSPACE"
        exit 1
    fi

    self=$(readlink -f $0)
    base=$(dirname $self)

    createrepo $WORKSPACE/packages/centos6

    exec docker run \
         --rm \
         --volume=$WORKSPACE/packages/centos6:/mnt \
         --volume=$(readlink -f $0):/root/run-test.sh \
         --volume=$base/common-test-packages.sh:/root/common-test.sh \
         --workdir=/mnt \
         centos:6 \
         /root/run-test.sh --install-scl
fi

if test "$1" = --install-scl ; then
    yum -q install --assumeyes scl-utils
    curl -L -O https://www.softwarecollections.org/en/scls/rhscl/python27/epel-6-x86_64/download/rhscl-python27-epel-6-x86_64.noarch.rpm
    yum -q install --assumeyes rhscl-python27-epel-6-x86_64.noarch.rpm
    yum -q install --assumeyes python27
    exec scl enable python27 $0
fi

cat >/etc/yum.repos.d/localrepo.repo <<EOF
[localrepo]
name=Arvados Test
baseurl=file:///mnt
gpgcheck=0
enabled=1
EOF

yum -q clean all
yum -q update
if ! yum -q install --assumeyes \
     python27-python-arvados-python-client python27-python-arvados-fuse arvados-node-manager
then
    exit 1
fi

mkdir -p /tmp/opts
cd /tmp/opts

for r in /mnt/python27-python-*x86_64.rpm ; do
    rpm2cpio $r | cpio -idm
done

for so in $(find . -name "*.so") ; do
    echo
    echo "== Packages dependencies for $so =="
    ldd $so | awk '($3 ~ /^\//){print $3}' | sort -u | xargs rpm -qf | sort -u
done

exec /root/common-test.sh
