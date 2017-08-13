// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// MultipageLoader retrieves a multi-page result set from the
// server. The constructor initiates the first page load.
//
// config.loadFunc is a function that accepts an array of
// paging-related filters, and returns a promise for the API
// response. loadFunc() must retrieve results in "modified_at desc"
// order.
//
// done is true if there are no more pages to load.
//
// loading is true if a network request is in progress.
//
// items is a stream that resolves to an array of all items retrieved so far.
//
// loadMore() loads the next page, if any.
window.models = window.models || {}
window.models.MultipageLoader = function(config) {
    var loader = this
    Object.assign(loader, config, {
        done: false,
        loading: false,
        items: m.stream(),
        thresholdItem: null,
        loadMore: function() {
            if (loader.done || loader.loading)
                return
            var filters = loader.thresholdItem ? [
                ["modified_at", "<=", loader.thresholdItem.modified_at],
                ["uuid", "!=", loader.thresholdItem.uuid],
            ] : []
            loader.loading = true
            loader.loadFunc(filters).then(function(resp) {
                var items = loader.items() || []
                Array.prototype.push.apply(items, resp.items)
                if (resp.items.length == 0)
                    loader.done = true
                else
                    loader.thresholdItem = resp.items[resp.items.length-1]
                loader.loading = false
                loader.items(items)
            }).catch(function(err) {
                loader.err = err
                loader.loading = false
            })
        },
    })
    loader.loadMore()
}

// MergingLoader merges results from multiple loaders (given in the
// config.children array) into a single result set.
//
// new MergingLoader({children: [loader, loader, ...]})
//
// The children must retrieve results in "modified_at desc" order.
window.models = window.models || {}
window.models.MergingLoader = function(config) {
    var loader = this
    Object.assign(loader, config, {
        // Sorted items ready to display, merged from all children.
        items: m.stream(),
        done: false,
        loading: false,
        loadable: function() {
            // Return an array of children that we could call
            // loadMore() on. Update loader.done and loader.loading.
            loader.done = true
            loader.loading = false
            return loader.children.filter(function(child) {
                if (child.done)
                    return false
                loader.done = false
                if (!child.loading)
                    return true
                loader.loading = true
                return false
            })
        },
        loadMore: function() {
            // Call loadMore() on children that have reached
            // lowWaterMark.
            loader.loadable().map(function(child) {
                if (child.items().length - child.itemsDisplayed < loader.lowWaterMark) {
                    loader.loading = true
                    child.loadMore()
                }
            })
        },
        mergeItems: function() {
            // cutoff is the topmost (recent) of {bottom (oldest) entry of
            // any child that still has more pages left to fetch}
            var cutoff
            loader.children.forEach(function(child) {
                var items = child.items()
                if (items.length == 0 || child.done)
                    return
                var last = items[items.length-1].modified_at
                if (!cutoff || cutoff < last)
                    cutoff = last
            })
            var combined = []
            loader.children.forEach(function(child) {
                child.itemsDisplayed = 0
                child.items().every(function(item) {
                    if (cutoff && item.modified_at < cutoff)
                        // Some other children haven't caught up to this
                        // point, so don't display this item or anything
                        // after it.
                        return false
                    combined.push(item)
                    child.itemsDisplayed++
                    return true // continue
                })
            })
            loader.items(combined.sort(function(a, b) {
                return a.modified_at < b.modified_at ? 1 : -1
            }))
        },
        // Number of undisplayed items to keep on hand for each result
        // set. When hitting "load more", if a result set already has
        // this many additional results available, we don't bother
        // fetching a new page. This is the _minimum_ number of rows
        // that will be added to loader.items in each "load more"
        // event (except for the case where all items are displayed).
        lowWaterMark: 23,
    })
    var childrenReady = m.stream.merge(loader.children.map(function(child) {
        return child.items
    }))
    childrenReady.map(loader.loadable)
    childrenReady.map(loader.mergeItems)
}
