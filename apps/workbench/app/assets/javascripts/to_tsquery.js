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

// to_tsquery_filters() converts a user-entered search query to a list of
// filters using the newly added (as for arvados 1.5) trigram indexes.
//
// Examples:
//
// "foo"     => [["any", "ilike", "%foo%"]]
// "foo.bar" => [["any", "ilike", "%foo.bar%"]]
// "foo bar" => [["any", "ilike", "%foo%"], ["any", "ilike", "%bar%"]]
// "foo|bar" => [["any", "ilike", "%foo%"], ["any", "ilike", "%bar%"]]
// ""        => []
// null      => []
window.to_tsquery_filters = function(q) {
    q = (q || '').replace(/[^-\w\.\/]+/g, ' ').trim()
    if (q == '')
        return []
    return q.split(" ").map(function(term) {
        return ["any", "ilike", "%"+term+"%"]
    })
}