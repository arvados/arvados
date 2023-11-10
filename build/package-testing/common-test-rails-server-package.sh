#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

if [ 0 = "$#" ]; then
    PACKAGE_NAME="$(basename "$0" | grep -Eo '\barvados.*$')"
    PACKAGE_NAME=${PACKAGE_NAME%.sh}
else
    PACKAGE_NAME=$1; shift
fi

cd "/var/www/${PACKAGE_NAME%-server}/current"

case "$TARGET" in
    debian*|ubuntu*)
        apt-get install -y nginx
        dpkg-reconfigure "$PACKAGE_NAME"
        ;;
    centos*)
        yum install --assumeyes httpd
        yum reinstall --assumeyes "$PACKAGE_NAME"
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

/usr/local/rvm/bin/rvm-exec default bundle list >"$ARV_PACKAGES_DIR/$PACKAGE_NAME.gems"
