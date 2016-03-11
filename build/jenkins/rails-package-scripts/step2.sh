#!/bin/sh
# This code runs after package variable definitions, before the actual
# pre/post package work, to set some variable and function defaults.

if [ -z "$INSTALL_PATH" ]; then
    cat >&2 <<EOF

PACKAGE BUILD ERROR: $0 is missing package metadata.

This package is buggy.  Please mail <support@curoverse.com> to let
us know the name and version number of the package you tried to
install, and we'll get it fixed.

EOF
    exit 3
fi

RELEASE_PATH=$INSTALL_PATH/current
RELEASE_CONFIG_PATH=$RELEASE_PATH/config
SHARED_PATH=$INSTALL_PATH/shared

RAILSPKG_SUPPORTS_CONFIG_CHECK=${RAILSPKG_SUPPORTS_CONFIG_CHECK:-1}
if ! type setup_extra_conffiles >/dev/null 2>&1; then
    setup_extra_conffiles() { return; }
fi
if ! type setup_before_nginx_restart >/dev/null 2>&1; then
    setup_before_nginx_restart() { return; }
fi
