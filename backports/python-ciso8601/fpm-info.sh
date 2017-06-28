# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

case "$TARGET" in
    centos*)
        fpm_depends+=(glibc)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(libc6)
        ;;
esac
