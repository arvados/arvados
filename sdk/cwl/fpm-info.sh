# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

fpm_depends+=(nodejs)

case "$TARGET" in
    debian12 | ubuntu2204 )
        fpm_depends+=(libcurl4)
        ;;

    debian* | ubuntu* )
        fpm_depends+=(libcurl4t64)
        ;;
esac

fpm_args+=(--conflicts=python-cwltool --conflicts=cwltool)
