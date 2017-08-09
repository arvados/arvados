// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.components = window.components || {}
window.components.collection_table_narrow = {
    view: function(vnode) {
        return m('table.table.table-condensed', [
            m('thead', m('tr', m('th', vnode.attrs.key))),
            m('tbody', [
                vnode.attrs.items().map(function(item) {
                    return m('tr', [
                        m('td', [
                            m('a', {href: '/collections/'+item.uuid}, item.name || '(unnamed)'),
                            m('br'),
                            item.modified_at,
                        ]),
                    ])
                }),
            ]),
        ])
    },
}

window.components.collection_search = {
    oninit: function(vnode) {
        vnode.state.sessionDB = new window.models.SessionDB()
        vnode.state.searchEntered = m.stream('')
        vnode.state.searchStart = m.stream('')
        vnode.state.items = {}
        vnode.state.searchStart.map(function(q) {
            var sessions = vnode.state.sessionDB.loadAll()
            var cookie = (new Date()).getTime()
            vnode.state.cookie = cookie
            Object.keys(sessions).map(function(key) {
                if (!vnode.state.items[key])
                    vnode.state.items[key] = m.stream([])
                vnode.state.sessionDB.request(sessions[key], 'arvados/v1/collections', {
                    data: {
                        filters: JSON.stringify(!q ? [] : [['any', '@@', q+':*']]),
                    },
                }).then(function(resp) {
                    if (cookie !== vnode.state.cookie)
                        // a newer query is in progress; ignore this result.
                        return
                    vnode.state.items[key](resp.items)
                })
            })
        })
    },
    view: function(vnode) {
        var items = vnode.state.items
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
                            oninput: m.withAttr('value', debounce(200, vnode.state.searchEntered)),
                        }),
                        m('.input-group-btn', [
                            m('input.btn.btn-primary[type=submit][value="Search"]'),
                        ]),
                    ]),
                ]),
                m('.col-md-6', [
                    'Searching sites: ',
                    Object.keys(items).length == 0
                        ? m('span.label.label-xs.label-danger', 'none')
                        : Object.keys(items).sort().map(function(key) {
                            return [m('span.label.label-xs.label-info', key), ' ']
                        }),
                    ' ',
                    m('a[href="/sessions"]', 'Add/remove sites'),
                ]),
            ]),
            m('.row', Object.keys(items).sort().map(function(key) {
                return m('.col-md-3', {key: key}, [
                    m(window.components.collection_table_narrow, {key: key, items: items[key]}),
                ])
            })),
        ])
    },
}

function debounce(t, f) {
    // Return a new function that waits until t milliseconds have
    // passed since it was last called, then calls f with its most
    // recent arguments.
    var this_was = this
    var pending
    return function() {
        var args = arguments
        if (pending) {
            console.log("debounce!")
            window.clearTimeout(pending)
        }
        pending = window.setTimeout(function() {
            pending = undefined
            f.apply(this_was, args)
        }, t)
    }
}
