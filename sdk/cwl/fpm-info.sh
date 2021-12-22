# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

fpm_depends+=(nodejs)

case "$TARGET" in
    debian* | ubuntu*)
        fpm_depends+=(libcurl3-gnutls python3-distutils)
        ;;
esac

fpm_args+=(--conflicts=python-cwltool --conflicts=cwltool)
