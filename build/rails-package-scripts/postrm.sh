#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This code runs after package variable definitions.

set -e

purge () {
  rm -rf $SHARED_PATH/vendor_bundle
  rm -rf $SHARED_PATH/log
  rm -rf $CONFIG_PATH
  rmdir $SHARED_PATH || true
  rmdir $INSTALL_PATH || true
}

if [ "$1" = 'purge' ]; then
  # This is a debian-based system and purge was requested
  purge
elif [ "$1" = "0" ]; then
  # This is an rpm-based system, no guarantees are made, always purge
  # Apparently yum doesn't actually remember what it installed.
  # Clean those files up here, then purge.
  rm -rf $RELEASE_PATH
  purge
fi
