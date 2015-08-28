#!/bin/sh

if test "$1" = --run-test ; then

    if test -z "$WORKSPACE" ; then
        echo "Must set WORKSPACE"
        exit 1
    fi

    self=$(readlink -f $0)
    cd $WORKSPACE/packages/debian7
    dpkg-scanpackages . /dev/null | gzip -c9 > Packages.gz

    exec docker run \
         --rm \
         --volume=$WORKSPACE/packages/$2:/mnt \
         --volume=$self:/root/run-test.sh \
         --workdir=/mnt \
         $3 \
         /root/run-test.sh
fi

echo "deb file:///mnt /" >>/etc/apt/sources.list
apt-get update
apt-get --assume-yes --force-yes install python-arvados-python-client python-arvados-fuse

python <<EOF
import arvados
import arvados_fuse
EOF
