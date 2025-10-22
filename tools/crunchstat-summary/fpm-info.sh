# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

case "$TARGET" in
    debian12 | ubuntu2204 )
        fpm_depends+=(libcurl4)
        ;;

    debian* | ubuntu* )
        fpm_depends+=(libcurl4t64)
        ;;
esac
