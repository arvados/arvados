# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

fpm_depends+=(fuse)

case "$TARGET" in
    centos*)
        fpm_depends+=(fuse-libs)
        ;;
esac
