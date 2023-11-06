# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

fpm_depends+=(ca-certificates)
case "$TARGET" in
    rocky*) fpm_depends+=(pam) ;;
esac

fpm_args+=(--conflicts=libpam-arvados)
