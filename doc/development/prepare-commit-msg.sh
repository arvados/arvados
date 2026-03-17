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
trailer="$(git interpret-trailers --trailer arvados </dev/null 2>/dev/null | grep @ || :)"

new_msg="$(mktemp --tmpdir="$(dirname "$msgfile")" commit-XXXXXX.txt)"
trap 'rm -f "$new_msg"' EXIT INT TERM QUIT
gawk -f - -v source="${1:-}" -v trailer="$trailer" -- "$msgfile" >"$new_msg" <<'EOF'
BEGIN { $0=trailer; trailer_key=$1; }
function write_trailer() {
  if (trailer) {
    if (last1 != trailer_key) { print ""; }
    print trailer;
    trailer="";
  }
}
END { write_trailer(); }
($0 == trailer) { trailer=""; }
((last1 == trailer_key && $1 != trailer_key) ||
 $1 == "#" || $1 == "---" || $1 == "diff") { write_trailer(); }
{ print; last1=$1; }
(NR == 1 && $1 == "Merge" && match($0, "['/]([0-9]+)-", ma)) {
    printf("\nRefs #%s.\n", ma[1]);
}
EOF
mv -f "$new_msg" "$msgfile"
