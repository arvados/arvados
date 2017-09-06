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
// state is:
// * 'loading' if a network request is in progress;
// * 'done' if there are no more items to load;
// * 'ready' otherwise.
//
// items is a stream that resolves to an array of all items retrieved so far.
//
// loadMore() loads the next page, if any.
window.MultipageLoader = function(config) {
    var loader = this
    Object.assign(loader, config, {
        state: 'ready',
        DONE: 'done',
        LOADING: 'loading',
        READY: 'ready',

        items: m.stream([]),
        thresholdItem: null,
        loadMore: function() {
            if (loader.state == loader.DONE || loader.state == loader.LOADING)
                return
            var filters = loader.thresholdItem ? [
                ["modified_at", "<=", loader.thresholdItem.modified_at],
                ["uuid", "!=", loader.thresholdItem.uuid],
            ] : []
            loader.state = loader.LOADING
            loader.loadFunc(filters).then(function(resp) {
                var items = loader.items()
                Array.prototype.push.apply(items, resp.items)
                if (resp.items.length == 0) {
                    loader.state = loader.DONE
                } else {
                    loader.thresholdItem = resp.items[resp.items.length-1]
                    loader.state = loader.READY
                }
                loader.items(items)
            }).catch(function(err) {
                loader.err = err
                loader.state = loader.READY
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
window.MergingLoader = function(config) {
    var loader = this
    Object.assign(loader, config, {
        // Sorted items ready to display, merged from all children.
        items: m.stream([]),
        state: 'ready',
        DONE: 'done',
        LOADING: 'loading',
        READY: 'ready',
        loadable: function() {
            // Return an array of children that we could call
            // loadMore() on. Update loader.state.
            loader.state = loader.DONE
            return loader.children.filter(function(child) {
                if (child.state == child.DONE)
                    return false
                if (child.state == child.LOADING) {
                    loader.state = loader.LOADING
                    return false
                }
                if (loader.state == loader.DONE)
                    loader.state = loader.READY
                return true
            })
        },
        loadMore: function() {
            // Call loadMore() on children that have reached
            // lowWaterMark.
            loader.loadable().map(function(child) {
                if (child.items().length - child.itemsDisplayed < loader.lowWaterMark) {
                    loader.state = loader.LOADING
                    child.loadMore()
                }
            })
        },
        mergeItems: function() {
            // We want to avoid moving items around on the screen once
            // they're displayed.
            //
            // To this end, here we find the last safely displayable
            // item ("cutoff") by getting the last item from each
            // unfinished child, and taking the topmost (most recent)
            // one of those.
            //
            // (If we were to display an item below that cutoff, the
            // next page of results from an unfinished child could
            // include items that get inserted above the cutoff,
            // causing the cutoff item to move down.)
            var cutoff
            var cutoffUnknown = false
            loader.children.forEach(function(child) {
                if (child.state == child.DONE)
                    return
                var items = child.items()
                if (items.length == 0) {
                    // No idea what's coming in the next page.
                    cutoffUnknown = true
                    return
                }
                var last = items[items.length-1].modified_at
                if (!cutoff || cutoff < last)
                    cutoff = last
            })
            if (cutoffUnknown)
                return
            var combined = []
            loader.children.forEach(function(child) {
                child.itemsDisplayed = 0
                child.items().every(function(item) {
                    if (cutoff && item.modified_at < cutoff)
                        // Don't display this item or anything after
                        // it (see "cutoff" comment above).
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
