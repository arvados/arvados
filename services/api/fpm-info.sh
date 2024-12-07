# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

fpm_depends+=(
    # Dependencies to build gems
    bison
    make
    "ruby >= 2.7.0"
    # Postinst script dependencies
    diffutils
    # Passenger dependencies
    procps
    # Dependencies of our API server code
    "git >= 1.7.10"
    shared-mime-info
)

case "$TARGET" in
    centos*|rocky*)
        fpm_depends+=(
            # Dependencies to build gems
            automake
            gcc-c++
            libcurl-devel
            postgresql
            postgresql-devel
            "ruby-devel >= 2.7.0"
            zlib-devel
            # Passenger runtime dependencies
            libnsl
        )
        ;;
    debian* | ubuntu*)
        fpm_depends+=(
            # Dependencies to build gems
            g++
            libcurl-ssl-dev
            libpq-dev
            postgresql-client
            "ruby-dev >= 2.7.0"
            zlib1g-dev
            # Passenger runtime dependencies
            libnsl2
            libnss-systemd
        )
        ;;
esac
