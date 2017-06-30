# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

case "$TARGET" in
    ubuntu1204)
        fpm_depends+=('libfuse2 = 2.9.2-5')
        ;;
esac
