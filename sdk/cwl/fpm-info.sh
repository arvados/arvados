# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

case "$TARGET" in
    debian8)
        fpm_depends+=(libgnutls-deb0-28 libcurl3-gnutls)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(libcurl3-gnutls)
        ;;
esac

fpm_args+=(--conflicts=python-cwltool --conflicts=cwltool)
