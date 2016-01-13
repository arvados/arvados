#!/bin/bash

set -eu

# Multiple .deb based distros symlink to this script, so extract the target
# from the invocation path.
target=$(echo $0 | sed 's/.*test-packages-\([^.]*\)\.sh.*/\1/')

apt-get -qq update
apt-get --assume-yes --force-yes install $1

mkdir -p /tmp/opts
cd /tmp/opts

export ARV_PACKAGES_DIR=/arvados/packages/$target

dpkg-deb -x $ARV_PACKAGES_DIR/"$1"_*.deb .

for so in $(find . -name "*.so") ; do
    echo
    echo "== Packages dependencies for $so =="
    ldd $so | awk '($3 ~ /^\//){print $3}' | sort -u | xargs dpkg -S | cut -d: -f1 | sort -u
done

exec /jenkins/common-test-packages.sh $1
