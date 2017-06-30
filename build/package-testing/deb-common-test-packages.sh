#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -eu

# Multiple .deb based distros symlink to this script, so extract the target
# from the invocation path.
target=$(echo $0 | sed 's/.*test-packages-\([^.]*\)\.sh.*/\1/')

export ARV_PACKAGES_DIR="/arvados/packages/$target"

dpkg-query --show > "$ARV_PACKAGES_DIR/$1.before"

apt-get -qq update
apt-get --assume-yes --allow-unauthenticated install "$1"

dpkg-query --show > "$ARV_PACKAGES_DIR/$1.after"

set +e
diff "$ARV_PACKAGES_DIR/$1.before" "$ARV_PACKAGES_DIR/$1.after" > "$ARV_PACKAGES_DIR/$1.diff"
set -e

mkdir -p /tmp/opts
cd /tmp/opts

export ARV_PACKAGES_DIR="/arvados/packages/$target"

dpkg-deb -x $(ls -t "$ARV_PACKAGES_DIR/$1"_*.deb | head -n1) .

while read so && [ -n "$so" ]; do
    echo
    echo "== Packages dependencies for $so =="
    ldd "$so" | awk '($3 ~ /^\//){print $3}' | sort -u | xargs dpkg -S | cut -d: -f1 | sort -u
done <<EOF
$(find -name '*.so')
EOF

exec /jenkins/package-testing/common-test-packages.sh "$1"
