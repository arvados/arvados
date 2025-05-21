#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

PACKAGE_NAME=arvados-api-server
API_GEMS_LS="$(mktemp --tmpdir api-gems-XXXXXX.list)"
trap 'rm -f "$API_GEMS_LS"' EXIT INT TERM QUIT

cd "/var/www/${PACKAGE_NAME%-server}"

cat_dropins() {
    cat /lib/systemd/system/arvados-railsapi.service.d/*.conf
}

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
    fi
    echo "Package $PACKAGE_NAME FAILED: \`grep" "$@" "\` returned exit code $actual_exit" >&2
    case "$actual_exit" in
        0) return 1 ;;
        *) return "$actual_exit" ;;
    esac
}

# This code is basically copy-pasted from build/rails-package-scripts/postinst.sh.
ruby_minor_ver="$(ruby -e 'puts RUBY_VERSION.split(".")[..1].join(".")')"
BUNDLE="$(gem contents --version '~> 2.4.0' bundler | grep -E '/(bin|exe)/bundle$' | tail -n1)"
if ! [ -x "$BUNDLE" ]; then
    # Some distros (at least Ubuntu 24.04) append the Ruby version to the
    # executable name, but that isn't reflected in the output of
    # `gem contents`. Check for that version.
    BUNDLE="$BUNDLE$ruby_minor_ver"
    if ! [ -x "$BUNDLE" ]; then
        echo "Package $PACKAGE_NAME ERROR: failed to find \`bundle\` command after package installation" >&2
        exit 11
    fi
fi
env -C current "$BUNDLE" list >"$ARV_PACKAGES_DIR/$PACKAGE_NAME.gems"
check_gem_dirs "initial install"

case "$TARGET" in
    debian*|ubuntu*)
        cat_dropins | expect_grep 0 -x SupplementaryGroups=www-data
        ;;
    rocky*)
        cat_dropins | expect_grep 1 "^SupplementaryGroups="
        microdnf --assumeyes install nginx
        microdnf --assumeyes reinstall "$PACKAGE_NAME" || test $? -eq 1
        check_gem_dirs "package reinstall"
        cat_dropins | expect_grep 0 -x SupplementaryGroups=nginx
        ;;
    *)
        echo "$0: WARNING: Unknown target '$TARGET'." >&2
        ;;
esac
