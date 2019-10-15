// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// to_tsquery_filters() converts a user-entered search query to a list of
// filters using the newly added (as for arvados 1.5) trigram indexes. It returns
// null if it can't come up with anything valid (e.g., q consists entirely of
// punctuation).
//
// Examples:
//
// "foo"     => [["any", "ilike", "%foo%"]]
// "foo.bar" => [["any", "ilike", "%foo.bar%"]]                         // "." is a word char in FT queries
// "foo/b-r" => [["any", "ilike", "%foo/b-r%"]]                         // "/" and "-", too
// "foo bar" => [["any", "ilike", "%foo%"], ["any", "ilike", "%bar%"]]
// "foo|bar" => [["any", "ilike", "%foo%"], ["any", "ilike", "%bar%"]]
// " oo|bar" => [["any", "ilike", "%oo%"], ["any", "ilike", "%bar%"]]
// ""        => []
// " "       => []
// null      => []
window.to_tsquery_filters = function(q) {
    q = (q || '').replace(/[^-\w\.\/]+/g, ' ').trim()
    if (q == '')
        return []
    return q.split(" ").map(function(term) {
        return ["any", "ilike", "%"+term+"%"]
    })
}