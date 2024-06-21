# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# We depend on the fuse package because arv-mount may run the `fusermount` tool.
fpm_depends+=(fuse)

case "$TARGET" in
    centos*|rocky*)
        # We depend on libfuse for llfuse.
        # We should declare a libcurl dependency, but it's a little academic
        # because rpm itself depends on it, so we can be pretty sure it's installed.
        fpm_depends+=(fuse-libs)
        ;;
    debian* | ubuntu*)
        # We depend on libfuse2 for llfuse.
        # We depend on libcurl because the Python SDK does for its Keep client.
        fpm_depends+=(libfuse2 libcurl3-gnutls)
        ;;
esac
