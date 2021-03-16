# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

fpm_depends+=('git >= 1.7.10')

case "$TARGET" in
    centos*)
        fpm_depends+=(libcurl-devel postgresql-devel bison make automake gcc gcc-c++ postgresql)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(libcurl-ssl-dev libpq-dev g++ bison zlib1g-dev make postgresql-client)
        ;;
esac
