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
                vnode.attrs.items.map(function(item) {
                    return m('tr', [
                        m('td', m('a.btn.btn-xs.btn-default', {href: item.session.baseURL.replace('://', '://workbench.')+'/collections/'+item.uuid}, 'Show')),
                        m('td', item.uuid),
                        m('td', item.name || '(unnamed)'),
                        m('td', m(window.components.datetime, {parse: item.modified_at})),
                    ])
                }),
            ]),
        ])
    },
}

function Pager(loadFunc) {
    // loadFunc(filters) returns a promise for a page of results.
    var pager = this
    Object.assign(pager, {
        done: false,
        items: m.stream(),
        lastModifiedAt: null,
        loadNextPage: function() {
            // Get the next page, if there are any more items to get.
            if (pager.done)
                return
            var filters = pager.lastModifiedAt ? [["modified_at", "<=", pager.lastModifiedAt]] : []
            loadFunc(filters).then(function(resp) {
                var items = pager.items() || []
                Array.prototype.push.apply(items, resp.items)
                if (resp.items.length == 0)
                    pager.done = true
                else
                    pager.lastModifiedAt = resp.items[resp.items.length-1].modified_at
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
        // items ready to display
        vnode.state.displayItems = m.stream([])
        // {sessionKey -> Pager}
        vnode.state.pagers = {}
        vnode.state.searchStart.map(function(q) {
            var sessions = vnode.state.sessionDB.loadAll()
            var cookie = (new Date()).getTime()
            var displayItems = m.stream([])
            vnode.state.displayItems = displayItems
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
                vnode.state.pagers[key] = pager
                pager.loadNextPage()
                return pager.items.map(function() { return key })
            })).map(function(keys) {
                var combined = []
                keys.forEach(function(key) {
                    vnode.state.pagers[key].items().forEach(function(item) {
                        item.session = sessions[key]
                        combined.push(item)
                    })
                })
                displayItems(combined.sort(function(a, b) {
                    return a.modified_at < b.modified_at ? 1 : -1
                }))
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
                                className: vnode.state.pagers[key].items() ? 'label-info' : 'label-default',
                            }, key), ' ']
                        }),
                    ' ',
                    m('a[href="/sessions"]', 'Add/remove sites'),
                ]),
            ]),
            m(window.components.collection_table_narrow, {
                items: vnode.state.displayItems(),
            }),
        ])
    },
}
