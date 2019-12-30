# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

LIBCLOUD_PIN=2.3.1.dev2

using_fork=true
if [[ $using_fork = true ]]; then
    LIBCLOUD_PIN_SRC="https://github.com/arvados/libcloud/archive/apache-libcloud-$LIBCLOUD_PIN.zip"
else
    LIBCLOUD_PIN_SRC=""
fi
