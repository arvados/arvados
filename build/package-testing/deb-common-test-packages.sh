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

apt-get $DASHQQ_UNLESS_DEBUG -y --allow-unauthenticated install "$1" >"$STDOUT_IF_DEBUG" 2>"$STDERR_IF_DEBUG" ||
    install_status=$?

dpkg-query --show > "$ARV_PACKAGES_DIR/$1.after"

diff "$ARV_PACKAGES_DIR/$1.before" "$ARV_PACKAGES_DIR/$1.after" > "$ARV_PACKAGES_DIR/$1.diff" || true

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
  find -type f -name '*.so' | while read so; do
      printf "\n== Package dependencies for %s ==\n" "$so"
      # dpkg is not fully aware of merged-/usr systems: ldd may list a library
      # under /lib where dpkg thinks it's under /usr/lib, or vice versa.
      # awk constructs globs that we pass to `dpkg --search` to be flexible
      # about which version we find. This could potentially return multiple
      # results, but doing better probably requires restructuring this whole
      # code to find and report the best match across multiple dpkg queries.
      ldd "$so" \
          | awk 'BEGIN { ORS="\0" } ($3 ~ /^\//) {print "*" $3}' \
          | sort --unique --zero-terminated \
          | xargs -0 --no-run-if-empty dpkg --search \
          | cut -d: -f1 \
          | sort --unique
  done
fi

case "${install_status:-0}-$1" in
    0-* | 100-arvados-api-server )
        exec /jenkins/package-testing/common-test-packages.sh "$1"
        ;;
    *)
        exit "$install_status"
        ;;
esac
