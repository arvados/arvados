// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// to_tsquery() converts a user-entered search query to a useful
// operand for the Arvados API "@@" filter. It returns null if it
// can't come up with anything valid (e.g., q consists entirely of
// punctuation).
//
// Examples:
//
// "foo"     => "foo:*"
// "foo_bar" => "foo:*&bar:*"
// "foo.bar" => "foo.bar:*"    // "." is a word char in FT queries
// "foo/b-r" => "foo/b-r:*"    // "/" and "-", too
// "foo|bar" => "foo:*&bar:*"
// " oo|ba " => "oo:*&ba:*"
// "__ "     => null
// ""        => null
// null      => null
window.to_tsquery = function(q) {
    q = (q || '').replace(/[^-\w\.\/]+/g, ' ').trim().replace(/ /g, ':*&')
    if (q == '')
        return null
    return q + ':*'
}
