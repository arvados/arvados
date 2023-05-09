#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
set -eu

# Set up
DEBUG=${ARVADOS_DEBUG:-0}
STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
DASHQQ_UNLESS_DEBUG=-qq
if [[ "$DEBUG" != "0" ]]; then
  STDOUT_IF_DEBUG=/dev/stdout
  STDERR_IF_DEBUG=/dev/stderr
  DASHQQ_UNLESS_DEBUG=
fi

# Multiple .deb based distros symlink to this script, so extract the target
# from the invocation path.
target=$(echo $0 | sed 's/.*test-packages-\([^.]*\)\.sh.*/\1/')

export ARV_PACKAGES_DIR="/arvados/packages/$target"

dpkg-query --show > "$ARV_PACKAGES_DIR/$1.before"

apt-get $DASHQQ_UNLESS_DEBUG --allow-insecure-repositories update

apt-get $DASHQQ_UNLESS_DEBUG -y --allow-unauthenticated install "$1" >"$STDOUT_IF_DEBUG" 2>"$STDERR_IF_DEBUG"

dpkg-query --show > "$ARV_PACKAGES_DIR/$1.after"

set +e
diff "$ARV_PACKAGES_DIR/$1.before" "$ARV_PACKAGES_DIR/$1.after" > "$ARV_PACKAGES_DIR/$1.diff"
set -e

mkdir -p /tmp/opts
cd /tmp/opts

export ARV_PACKAGES_DIR="/arvados/packages/$target"

if [[ -f $(ls -t "$ARV_PACKAGES_DIR/$1"_*.deb 2>/dev/null | head -n1) ]] ; then
    debpkg=$(ls -t "$ARV_PACKAGES_DIR/$1"_*.deb | head -n1)
else
    debpkg=$(ls -t "$ARV_PACKAGES_DIR/processed/$1"_*.deb | head -n1)
fi

dpkg-deb -x $debpkg .

if [[ "$DEBUG" != "0" ]]; then
  while read so && [ -n "$so" ]; do
      echo
      echo "== Packages dependencies for $so =="
      ldd "$so" | awk '($3 ~ /^\//){print $3}' | sort -u | xargs -r dpkg -S | cut -d: -f1 | sort -u
  done <<EOF
$(find -name '*.so')
EOF
fi

exec /jenkins/package-testing/common-test-packages.sh "$1"
