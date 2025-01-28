#!/bin/sh
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

set -e
set -u

usage() {
    cat <<EOF
usage: $0 SUBCOMMAND [options ...]

Subcommands:
  $0 enable [extension]
EOF
}

# Enumerate GPU devices on the host and output a standard "driver" name for
# each one found. Currently only detects `nvidia`.
detect_gpus() {
    # -vmm sets a machine-readable output format.
    # The -d option queries 3D controllers only.
    # You must stick with standard awk - no GNU extensions.
    lspci -vmm -d ::0302 | awk '
BEGIN { FS="\t"; }
($1 != "Vendor:") { next; }
(tolower($2) ~ /^nvidia/) { print "nvidia"; }
'
}

case "${1:-}" in
    "-?"|-h|--help|help)
        usage
        ;;

    enable)
        src_ext="${2:-avail}"
        # Ensure src_ext starts with a dot
        src_ext=".${src_ext#.}"
        dst_dir=/run/modules-load.d
        mkdir -p "$dst_dir"
        detect_gpus | while read driver; do
            src="/etc/modules-load.d/$driver$src_ext"
            if [ -e "$src" ]; then
                ln -s "$src" "$dst_dir/$driver.conf"
            fi
        done
        ;;

    *)
        echo "$0: unknown subcommand: ${1:-}" >&2
        usage >&2
        exit 2
        ;;
esac
