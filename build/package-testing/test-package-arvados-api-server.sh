#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

PACKAGE_NAME=arvados-api-server
API_GEMS_LS="$(mktemp --tmpdir api-gems-XXXXXX.list)"
trap 'rm -f "$API_GEMS_LS"' EXIT INT TERM QUIT

cd "/var/www/${PACKAGE_NAME%-server}"

check_gem_dirs() {
    local when="$1"; shift
    env -C shared/vendor_bundle/ruby ls -1 >"$API_GEMS_LS"
    local ls_count="$(wc -l <"$API_GEMS_LS")"
    if [ "$ls_count" = 1 ]; then
        return 0
    fi
    echo "Package $PACKAGE_NAME FAILED: $ls_count gem directories created after $when:" >&2
    case "${ARVADOS_DEBUG:-0}" in
        0) cat "$API_GEMS_LS" >&2 ;;
        *) env -C shared/vendor_bundle/ruby find -maxdepth 3 -type d -ls >&2 ;;
    esac
    return 11
}

check_gem_dirs "initial install"

case "$TARGET" in
    debian*|ubuntu*)
        apt-get install -y nginx
        dpkg-reconfigure "$PACKAGE_NAME"
        ;;
    rocky*)
        microdnf --assumeyes install httpd
        microdnf --assumeyes reinstall "$PACKAGE_NAME"
        ;;
    *)
        echo -e "$0: Unknown target '$TARGET'.\n" >&2
        exit 1
        ;;
esac

check_gem_dirs "package reinstall"
env -C current bundle list >"$ARV_PACKAGES_DIR/$PACKAGE_NAME.gems"
