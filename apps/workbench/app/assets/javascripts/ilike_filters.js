// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// ilike_filters() converts a user-entered search query to a list of
// filters using the newly added (as of Arvados 1.5) trigram indexes. It returns
// [] (empty list) if it can't come up with anything valid (e.g., q consists
// entirely of punctuation).
//
// Examples:
//
// "foo"     => [["any", "ilike", "%foo%"]]
// "foo.bar" => [["any", "ilike", "%foo.bar%"]]                         // "." is a word char in ilike queries
// "foo/b-r" => [["any", "ilike", "%foo/b-r%"]]                         // "/" and "-", too
// "foo bar" => [["any", "ilike", "%foo%"], ["any", "ilike", "%bar%"]]
// "foo|bar" => [["any", "ilike", "%foo%"], ["any", "ilike", "%bar%"]]
// " oo|bar" => [["any", "ilike", "%oo%"], ["any", "ilike", "%bar%"]]
// ""        => []
// " "       => []
// null      => []
window.ilike_filters = function(q) {
    q = (q || '').replace(/[^-\w\.\/]+/g, ' ').trim()
    if (q == '')
        return []
    return q.split(" ").map(function(term) {
        return ["any", "ilike", "%"+term+"%"]
    })
}