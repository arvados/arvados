#!/bin/sh

if test "$1" = --run-test ; then

    if test -z "$WORKSPACE" ; then
        echo "Must set WORKSPACE"
        exit 1
    fi

    self=$(readlink -f $0)
    base=$(dirname $self)

    cd $WORKSPACE/packages/$2
    dpkg-scanpackages . /dev/null | gzip -c9 > Packages.gz

    exec docker run \
         --rm \
         --volume=$WORKSPACE/packages/$2:/mnt \
         --volume=$self:/root/run-test.sh \
         --volume=$base/common-test-packages.sh:/root/common-test.sh \
         --workdir=/mnt \
         $3 \
         /root/run-test.sh
fi

echo "deb file:///mnt /" >>/etc/apt/sources.list
apt-get -qq update
if ! apt-get -qq --assume-yes --force-yes install python-arvados-python-client python-arvados-fuse ; then
    exit 1
fi

mkdir -p /tmp/opts
cd /tmp/opts

for r in /mnt/python-*amd64.deb ; do
    dpkg-deb -x $r .
done

for so in $(find . -name "*.so") ; do
    echo
    echo "== Packages dependencies for $so =="
    ldd $so | awk '($3 ~ /^\//){print $3}' | sort -u | xargs dpkg -S | cut -d: -f1 | sort -u
done

exec /root/common-test.sh
