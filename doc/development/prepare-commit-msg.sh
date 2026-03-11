#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This Git hook adds refs to branch merges and adds the Arvados DCO sign-off
# if you have configured it to do so.

set -e
set -u

msgfile="$1"; shift

case "${1:-}" in
    merge)
        new_msg="$(mktemp --tmpdir="$(dirname "$msgfile")" commit-XXXXXX.txt)"
        trap 'rm -f "$new_msg"' EXIT INT TERM QUIT
        gawk -f - -- "$msgfile" >"$new_msg" <<'EOF'
{ print; }
(NR == 1 && $1 == "Merge" && match($0, "['/]([0-9]+)-", ma)) {
    printf("\nRefs #%s.\n", ma[1]);
}
EOF
        mv -f "$new_msg" "$msgfile"
        ;;
esac

if git config trailer.arvados.cmd >/dev/null; then
    git interpret-trailers --in-place --trailer arvados "$msgfile"
fi
exit 0
