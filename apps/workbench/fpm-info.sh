# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

case "$TARGET" in
    centos*)
        fpm_depends+=(git)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(git g++)
        ;;
esac
