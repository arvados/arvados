#!/bin/bash

# Multiple .deb based distros symlink to this script, so extract the target
# from the invocation path.
target=$(echo $0 | sed 's/.*test-packages-\([^.]*\)\.sh.*/\1/')

apt-get -qq update
if ! apt-get --assume-yes --force-yes install \
     python-arvados-python-client python-arvados-fuse arvados-node-manager
then
    exit 1
fi

mkdir -p /tmp/opts
cd /tmp/opts

for r in /arvados/packages/$target/python-*amd64.deb ; do
    dpkg-deb -x $r .
done

for so in $(find . -name "*.so") ; do
    echo
    echo "== Packages dependencies for $so =="
    ldd $so | awk '($3 ~ /^\//){print $3}' | sort -u | xargs dpkg -S | cut -d: -f1 | sort -u
done

exec /jenkins/common-test-packages.sh
