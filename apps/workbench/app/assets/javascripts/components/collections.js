// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.components = window.components || {}
window.components.collection_table_narrow = {
    view: function(vnode) {
        return m('table.table.table-condensed', [
            m('thead', m('tr', [
                m('th'),
                m('th', 'uuid'),
                m('th', 'name'),
                m('th', 'last modified'),
            ])),
            m('tbody', [
                vnode.attrs.results.displayable.map(function(item) {
                    return m('tr', [
                        m('td', m('a.btn.btn-xs.btn-default', {href: item.session.baseURL.replace('://', '://workbench.')+'/collections/'+item.uuid}, 'Show')),
                        m('td', item.uuid),
                        m('td', item.name || '(unnamed)'),
                        m('td', m(window.components.datetime, {parse: item.modified_at})),
                    ])
                }),
            ]),
            m('tfoot', m('tr', [
                m('th[colspan=4]', m('button.btn.btn-xs', {
                    className: vnode.attrs.results.loadMore ? 'btn-primary' : 'btn-default',
                    style: {
                        display: 'block',
                        width: '12em',
                        marginLeft: 'auto',
                        marginRight: 'auto',
                    },
                    disabled: !vnode.attrs.results.loadMore,
                    onclick: function() {
                        vnode.attrs.results.loadMore()
                        return false
                    },
                }, vnode.attrs.results.loadMore ? 'Load more' : '(loading)')),
            ])),
        ])
    },
}

function Pager(loadFunc) {
    // loadFunc(filters) returns a promise for a page of results.
    var pager = this
    Object.assign(pager, {
        done: false,
        items: m.stream(),
        thresholdItem: null,
        loadNextPage: function() {
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

window.components.collection_search = {
    oninit: function(vnode) {
        vnode.state.sessionDB = new window.models.SessionDB()
        vnode.state.searchEntered = m.stream('')
        vnode.state.searchStart = m.stream('')
        vnode.state.searchStart.map(function(q) {
            var sessions = vnode.state.sessionDB.loadAll()
            var cookie = (new Date()).getTime()
            // Each time searchStart() is called we replace the
            // vnode.state.results stream with a new one, and use
            // the local variable to update results in callbacks. This
            // avoids crosstalk between AJAX calls from consecutive
            // searches.
            var results = {
                // Sorted items ready to display, merged from all
                // pagers.
                displayable: [],
                pagers: {},
                loadMore: false,
                // Number of undisplayed items to keep on hand for
                // each result set. When hitting "load more", if a
                // result set already has this many additional results
                // available, we don't bother fetching a new
                // page. This is the _minimum_ number of rows that
                // will be added to results.displayable in each "load
                // more" event (except for the case where all items
                // are displayed).
                lowWaterMark: 23,
            }
            vnode.state.results = results
            m.stream.merge(Object.keys(sessions).map(function(key) {
                var pager = new Pager(function(filters) {
                    if (q)
                        filters.push(['any', '@@', q+':*'])
                    return vnode.state.sessionDB.request(sessions[key], 'arvados/v1/collections', {
                        data: {
                            filters: JSON.stringify(filters),
                            count: 'none',
                        },
                    })
                })
                results.pagers[key] = pager
                pager.loadNextPage()
                // Resolve the stream with the session key when the
                // results arrive.
                return pager.items.map(function() { return key })
            })).map(function(keys) {
                // Top (most recent) of {bottom (oldest) entry of any
                // pager that still has more pages left to fetch}
                var cutoff
                keys.forEach(function(key) {
                    var pager = results.pagers[key]
                    var items = pager.items()
                    if (items.length == 0 || pager.done)
                        return
                    var last = items[items.length-1].modified_at
                    if (!cutoff || cutoff < last)
                        cutoff = last
                })
                var combined = []
                keys.forEach(function(key) {
                    var pager = results.pagers[key]
                    pager.itemsDisplayed = 0
                    pager.items().every(function(item) {
                        if (cutoff && item.modified_at < cutoff)
                            // Some other pagers haven't caught up to
                            // this point, so don't display this item
                            // or anything after it.
                            return false
                        item.session = sessions[key]
                        combined.push(item)
                        pager.itemsDisplayed++
                        return true // continue
                    })
                })
                results.displayable = combined.sort(function(a, b) {
                    return a.modified_at < b.modified_at ? 1 : -1
                })
                // Make a new loadMore function that hits the pagers
                // (if necessary according to lowWaterMark)... or set
                // results.loadMore to false if there is nothing left
                // to fetch.
                var loadable = []
                Object.keys(results.pagers).map(function(key) {
                    if (!results.pagers[key].done)
                        loadable.push(results.pagers[key])
                })
                if (loadable.length == 0)
                    results.loadMore = false
                else
                    results.loadMore = function() {
                        results.loadMore = false
                        loadable.map(function(pager) {
                            if (pager.items().length - pager.itemsDisplayed < results.lowWaterMark)
                                pager.loadNextPage()
                        })
                    }
            })
        })
    },
    view: function(vnode) {
        var sessions = vnode.state.sessionDB.loadAll()
        return m('form', {
            onsubmit: function() {
                vnode.state.searchStart(vnode.state.searchEntered())
                return false
            },
        }, [
            m('.row', [
                m('.col-md-6', [
                    m('.input-group', [
                        m('input#search.form-control[placeholder=Search]', {
                            oninput: m.withAttr('value', vnode.state.searchEntered),
                        }),
                        m('.input-group-btn', [
                            m('input.btn.btn-primary[type=submit][value="Search"]'),
                        ]),
                    ]),
                ]),
                m('.col-md-6', [
                    'Searching sites: ',
                    Object.keys(sessions).length == 0
                        ? m('span.label.label-xs.label-danger', 'none')
                        : Object.keys(sessions).sort().map(function(key) {
                            return [m('span.label.label-xs', {
                                className: vnode.state.results.pagers[key].items() ? 'label-info' : 'label-default',
                            }, key), ' ']
                        }),
                    ' ',
                    m('a[href="/sessions"]', 'Add/remove sites'),
                ]),
            ]),
            m(window.components.collection_table_narrow, {
                results: vnode.state.results,
            }),
        ])
    },
}
