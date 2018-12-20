#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

# Detect rpm-based systems: the exit code of the following command is zero
# on rpm-based systems
if /usr/bin/rpm -q -f /usr/bin/rpm >/dev/null 2>&1; then
    # Red Hat ("%{...}" is interpolated at package build time)
    pkg="%{name}"
    pkgtype=rpm
    prefix="${RPM_INSTALL_PREFIX}"
else
    # Debian
    script="$(basename "${0}")"
    pkg="${script%.prerm}"
    pkgtype=deb
    prefix=/usr
fi

# populated from the build script
# dash only supports one array, $@
if [ "%FPM_BINARIES" != "" ]; then
  set %FPM_BINARIES
fi

if [ "$pkg" != "" ]; then
  # Remove the binary python files so the package manager doesn't throw warnings
  # on removing the package.
  find /usr/share/%PYTHON/dist/$pkg -iname *.pyc -exec rm {} \; || true
  find /usr/share/%PYTHON/dist/$pkg -iname *.pyo -exec rm {} \; || true

  if [ "%FPM_BINARIES" != "" ]; then
    # read from $@
    for binary do
      if [ -L /usr/bin/$binary ]; then
        # Remove the symlinks we installed
        rm -f /usr/bin/$binary
      fi
    done
  fi

  if [ "${pkg#python-}" = "arvados-cwl-runner" ]; then
    if [ -L /usr/bin/cwl-runner ]; then
      rm -f /usr/bin/cwl-runner
    fi
  fi
fi
