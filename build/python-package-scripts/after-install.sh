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
    pkg="${script%.postinst}"
    pkgtype=deb
    prefix=/usr
fi

# populated from the build script
# dash only supports one array, $@
if [ "%FPM_BINARIES" != "" ]; then
  set %FPM_BINARIES
fi

# Install symlinks to the binary/binaries
if [ "$pkg" != "" ]; then

  if [ "%FPM_BINARIES" != "" ]; then
    # read from $@
    for binary do
      if [ -e /usr/bin/$binary ]; then
        rm -f /usr/bin/$binary
      fi
       ln -s /usr/share/%PYTHON/dist/$pkg/bin/$binary /usr/bin/$binary
    done
  fi

  # special case for arvados-cwl-runner
  if [ "${pkg#python-}" = "arvados-cwl-runner" ]; then
    if [ -e /usr/bin/cwl-runner ]; then
      rm -f /usr/bin/cwl-runner
    fi
    ln -s /usr/share/%PYTHON/dist/$pkg/bin/$binary /usr/bin/cwl-runner
  fi
fi

