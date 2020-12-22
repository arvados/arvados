# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

case "$TARGET" in
    ubuntu1604)
        fpm_depends+=()
        ;;
    debian* | ubuntu*)
        fpm_depends+=(python3-distutils)
        ;;
esac
