#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This code runs after package variable definitions, before the actual
# pre/post package work, to set some variable and function defaults.

if [ -z "$INSTALL_PATH" ]; then
    cat >&2 <<EOF

PACKAGE BUILD ERROR: $0 is missing package metadata.

This package is buggy.  Please mail <packaging@arvados.org> to let
us know the name and version number of the package you tried to
install, and we'll get it fixed.

EOF
    exit 3
fi

RELEASE_PATH=$INSTALL_PATH/current
RELEASE_CONFIG_PATH=$RELEASE_PATH/config
SHARED_PATH=$INSTALL_PATH/shared

if ! type setup_extra_conffiles >/dev/null 2>&1; then
    setup_extra_conffiles() { return; }
fi
if ! type setup_before_nginx_restart >/dev/null 2>&1; then
    setup_before_nginx_restart() { return; }
fi

if [ -e /run/systemd/system ]; then
    USING_SYSTEMD=1
else
    USING_SYSTEMD=0
fi

if which service >/dev/null 2>&1; then
    USING_SERVICE=1
else
    USING_SERVICE=0
fi

guess_service_manager() {
    if [ 1 = "$USING_SYSTEMD" ]; then
        echo systemd
    elif [ 1 = "$USING_SERVICE" ]; then
        echo service
    else
        return 1
    fi
}

list_services_systemd() {
    test 1 = "$USING_SYSTEMD" || return
    # Print only service names, without the `.service` suffix.
    systemctl list-unit-files '*.service' \
        | awk '($1 ~ /\.service/){print substr($1, 1, length($1) - 8)}'
}

list_services_service() {
    test 1 = "$USING_SERVICE" || return
    # Output is completely different across Debian and Red Hat.
    # We can't really parse it.
    service --status-all 2>/dev/null
}

service_command() {
    local service_manager="$1"; shift
    local command="$1"; shift
    local service="$1"; shift
    case "$service_manager" in
        systemd) systemctl "$command" "$service" ;;
        service) service "$service" "$command" ;;
    esac
}

if ! guess_service_manager >/dev/null; then
    echo "WARNING: Unsupported init system. Can't manage web service." >&2
fi
