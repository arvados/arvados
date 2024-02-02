#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -eu

# Set up
DEBUG=${ARVADOS_DEBUG:-0}
STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
if [[ "$DEBUG" != "0" ]]; then
  STDOUT_IF_DEBUG=/dev/stdout
  STDERR_IF_DEBUG=/dev/stderr
fi

target="$(basename "$0" .sh)"
target="${target##*-}"

microdnf --assumeyes clean all
touch /var/lib/rpm/*

export ARV_PACKAGES_DIR="/arvados/packages/$target"

rpm -qa | sort > "$ARV_PACKAGES_DIR/$1.before"
microdnf --assumeyes install "$1"
rpm -qa | sort > "$ARV_PACKAGES_DIR/$1.after"
diff "$ARV_PACKAGES_DIR/$1".{before,after} >"$ARV_PACKAGES_DIR/$1.diff" || true

mkdir -p /tmp/opts
cd /tmp/opts

rpm2cpio $(ls -t "$ARV_PACKAGES_DIR/$1"-*.rpm | head -n1) | cpio -idm 2>/dev/null

if [[ "$DEBUG" != "0" ]]; then
  find -name '*.so' | while read so; do
      echo -e "\n== Packages dependencies for $so =="
      ldd "$so" \
          | awk '($3 ~ /^\//){print $3}' | sort -u | xargs rpm -qf | sort -u
  done
fi

exec /jenkins/package-testing/common-test-packages.sh "$1"
