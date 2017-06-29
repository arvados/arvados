# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

case "$TARGET" in
    centos*)
        build_depends+=('fuse-devel')
        fpm_depends+=(glibc fuse-libs)
        ;;
    ubuntu1204)
        build_depends+=(libfuse2 libfuse-dev)
        fpm_depends+=(libc6 python-contextlib2 'libfuse2 = 2.9.2-5' 'fuse = 2.9.2-5')
        ;;
    debian* | ubuntu*)
        build_depends+=('libfuse-dev')
        fpm_depends+=(libc6 'libfuse2 > 2.9.0' 'fuse > 2.9.0')
        ;;
esac
