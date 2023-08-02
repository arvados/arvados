# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

case "$TARGET" in
    centos*|rocky*)
        fpm_depends+=(mailcap)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(mime-support)
        ;;
esac
