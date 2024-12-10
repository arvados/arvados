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

expect_grep() {
    local expect_exit="$1"; shift
    local actual_exit=0
    grep "$@" >/dev/null || actual_exit=$?
    if [ "$actual_exit" -eq "$expect_exit" ]; then
        return 0
    elif [ "$actual_exit" -eq 0 ]; then
        return 1
    else
        return "$actual_exit"
    fi
}

env -C current bundle list >"$ARV_PACKAGES_DIR/$PACKAGE_NAME.gems"
check_gem_dirs "initial install"

SVC_OVERRIDES="$(mktemp --tmpdir arvados-railsapi-XXXXXX.conf)"
trap 'rm -f "$API_GEMS_LS" "$SVC_OVERRIDES"' EXIT INT TERM QUIT
cat /lib/systemd/system/arvados-railsapi.service.d/*.conf >"$SVC_OVERRIDES"

case "$TARGET" in
    debian*|ubuntu*)
        expect_grep 0 -x SupplementaryGroups=www-data "$SVC_OVERRIDES"
        ;;
    rocky*)
        expect_grep 1 "^SupplementaryGroups=" "$SVC_OVERRIDES"
        microdnf --assumeyes install nginx
        microdnf --assumeyes reinstall "$PACKAGE_NAME"
        check_gem_dirs "package reinstall"
        expect_grep 0 -x SupplementaryGroups=nginx "$SVC_OVERRIDES"
        ;;
    *)
        echo "$0: WARNING: Unknown target '$TARGET'." >&2
        ;;
esac
