# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

case "$TARGET" in
    centos*)
        fpm_depends+=(git bison make automake gcc gcc-c++ graphviz)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(git g++ bison zlib1g-dev make graphviz)
        ;;
esac
