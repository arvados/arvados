# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

case "$TARGET" in
    centos*|rocky*)
        fpm_depends+=(fuse-libs)
        ;;

    debian12 | ubuntu2204 )
        fpm_depends+=(libfuse2)
        ;;

    debian* | ubuntu* )
        fpm_depends+=(libfuse2t64)
        ;;
esac
