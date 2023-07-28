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

case "$target" in
    centos*) yum -q clean all ;;
    rocky*) microdnf clean all ;;
esac
touch /var/lib/rpm/*

export ARV_PACKAGES_DIR="/arvados/packages/$target"

rpm -qa | sort > "$ARV_PACKAGES_DIR/$1.before"

case "$target" in
    centos*) yum install --assumeyes -e 0 $1 ;;
    rocky*) microdnf install $1 ;;
esac

rpm -qa | sort > "$ARV_PACKAGES_DIR/$1.after"

diff "$ARV_PACKAGES_DIR/$1".{before,after} >"$ARV_PACKAGES_DIR/$1.diff" || true

# Enable any Software Collections that the package depended on.
if [[ -d /opt/rh ]]; then
    # We have to stage the list to a file, because `ls | while read` would
    # make a subshell, causing the `source` lines to have no effect.
    scl_list=$(mktemp)
    ls /opt/rh >"$scl_list"

    # SCL scripts aren't designed to run with -eu.
    set +eu
    while read scl; do
        source scl_source enable "$scl"
    done <"$scl_list"
    set -eu
    rm "$scl_list"
fi

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
