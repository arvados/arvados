# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

case "$TARGET" in
    debian8)
        fpm_depends+=(libgnutls-deb0-28 libcurl3-gnutls)
        ;;
    debian9 | ubuntu1604)
        fpm_depends+=(libcurl3-gnutls libpython2.7)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(libcurl3-gnutls libpython2.7 python3-distutils)
        ;;
esac

fpm_args+=(--conflicts=python-cwltool --conflicts=cwltool)
