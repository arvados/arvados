// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.models = window.models || {}
window.models.Pager = function(loadFunc) {
    // loadFunc(filters) must return a promise for a page of results.
    var pager = this
    Object.assign(pager, {
        done: false,
        items: m.stream(),
        thresholdItem: null,
        loadMore: function() {
            // Get the next page, if there are any more items to get.
            if (pager.done)
                return
            var filters = pager.thresholdItem ? [
                ["modified_at", "<=", pager.thresholdItem.modified_at],
                ["uuid", "!=", pager.thresholdItem.uuid],
            ] : []
            loadFunc(filters).then(function(resp) {
                var items = pager.items() || []
                Array.prototype.push.apply(items, resp.items)
                if (resp.items.length == 0)
                    pager.done = true
                else
                    pager.thresholdItem = resp.items[resp.items.length-1]
                pager.items(items)
            })
        },
    })
}

// MultisiteLoader loads pages of results from multiple API sessions
// and merges them into a single result set.
//
// The constructor implicitly starts an initial page load for each
// session.
//
// new MultisiteLoader({loadFunc: function(session, filters){...},
// sessionDB: new window.models.SessionDB()}
//
// At any given time, ml.loadMore will be either false (meaning a page
// load is in progress or there are no more results to fetch) or a
// function that starts loading more results.
//
// loadFunc() must retrieve results in "modified_at desc" order.
window.models = window.models || {}
window.models.MultisiteLoader = function(config) {
    var loader = this
    if (!(config.loadFunc && config.sessionDB))
        throw new Error("MultisiteLoader constructor requires loadFunc and sessionDB")
    Object.assign(loader, config, {
        // Sorted items ready to display, merged from all pagers.
        displayable: [],
        done: false,
        pagers: {},
        loadMore: false,
        // Number of undisplayed items to keep on hand for each result
        // set. When hitting "load more", if a result set already has
        // this many additional results available, we don't bother
        // fetching a new page. This is the _minimum_ number of rows
        // that will be added to loader.displayable in each "load
        // more" event (except for the case where all items are
        // displayed).
        lowWaterMark: 23,
    })
    var sessions = loader.sessionDB.loadAll()
    m.stream.merge(Object.keys(sessions).map(function(key) {
        var pager = new window.models.Pager(loader.loadFunc.bind(null, sessions[key]))
        loader.pagers[key] = pager
        pager.loadMore()
        // Resolve the stream with the session key when the results
        // arrive.
        return pager.items.map(function() { return key })
    })).map(function(keys) {
        // Top (most recent) of {bottom (oldest) entry of any pager
        // that still has more pages left to fetch}
        var cutoff
        keys.forEach(function(key) {
            var pager = loader.pagers[key]
            var items = pager.items()
            if (items.length == 0 || pager.done)
                return
            var last = items[items.length-1].modified_at
            if (!cutoff || cutoff < last)
                cutoff = last
        })
        var combined = []
        keys.forEach(function(key) {
            var pager = loader.pagers[key]
            pager.itemsDisplayed = 0
            pager.items().every(function(item) {
                if (cutoff && item.modified_at < cutoff)
                    // Some other pagers haven't caught up to this
                    // point, so don't display this item or anything
                    // after it.
                    return false
                item.session = sessions[key]
                combined.push(item)
                pager.itemsDisplayed++
                return true // continue
            })
        })
        loader.displayable = combined.sort(function(a, b) {
            return a.modified_at < b.modified_at ? 1 : -1
        })
        // Make a new loadMore function that hits the pagers (if
        // necessary according to lowWaterMark)... or set
        // loader.loadMore to false if there is nothing left to fetch.
        var loadable = []
        Object.keys(loader.pagers).map(function(key) {
            if (!loader.pagers[key].done)
                loadable.push(loader.pagers[key])
        })
        if (loadable.length == 0) {
            loader.done = true
            loader.loadMore = false
        } else
            loader.loadMore = function() {
                loader.loadMore = false
                loadable.map(function(pager) {
                    if (pager.items().length - pager.itemsDisplayed < loader.lowWaterMark)
                        pager.loadMore()
                })
            }
    })
}
