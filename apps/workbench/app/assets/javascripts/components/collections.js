// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.components = window.components || {}
window.components.collection_table = {
    maybeLoadMore: function(dom) {
        var loader = this.loader
        if (loader.done || !loader.loadMore)
            // Can't load more content anyway: no point in
            // checking anything else.
            return
        var contentRect = dom.getBoundingClientRect()
        var scroller = window // TODO: use dom's nearest ancestor with scrollbars
        if (contentRect.bottom < 2 * scroller.innerHeight) {
            // We have less than 1 page worth of content available
            // below the visible area. Load more.
            loader.loadMore()
            // Indicate loading is in progress.
            window.requestAnimationFrame(m.redraw)
        }
    },
    oncreate: function(vnode) {
        vnode.state.maybeLoadMore = vnode.state.maybeLoadMore.bind(vnode.state, vnode.dom)
        window.addEventListener('scroll', vnode.state.maybeLoadMore)
        window.addEventListener('resize', vnode.state.maybeLoadMore)
        vnode.state.timer = window.setInterval(vnode.state.maybeLoadMore, 200)
        vnode.state.onupdate(vnode)
    },
    onupdate: function(vnode) {
        vnode.state.loader = vnode.attrs.loader
    },
    onremove: function(vnode) {
        window.clearInterval(vnode.state.timer)
        window.removeEventListener('scroll', vnode.state.maybeLoadMore)
        window.removeEventListener('resize', vnode.state.maybeLoadMore)
    },
    view: function(vnode) {
        return m('table.table.table-condensed', [
            m('thead', m('tr', [
                m('th'),
                m('th', 'uuid'),
                m('th', 'name'),
                m('th', 'last modified'),
            ])),
            m('tbody', [
                vnode.attrs.loader.displayable.map(function(item) {
                    return m('tr', [
                        m('td', m('a.btn.btn-xs.btn-default', {href: item.session.baseURL.replace('://', '://workbench.')+'collections/'+item.uuid}, 'Show')),
                        m('td.arvados-uuid', item.uuid),
                        m('td', item.name || '(unnamed)'),
                        m('td', m(window.components.datetime, {parse: item.modified_at})),
                    ])
                }),
            ]),
            m('tfoot', m('tr', [
                vnode.attrs.loader.done ? null : m('th[colspan=4]', m('button.btn.btn-xs', {
                    className: vnode.attrs.loader.loadMore ? 'btn-primary' : 'btn-default',
                    style: {
                        display: 'block',
                        width: '12em',
                        marginLeft: 'auto',
                        marginRight: 'auto',
                    },
                    disabled: !vnode.attrs.loader.loadMore,
                    onclick: function() {
                        vnode.attrs.loader.loadMore()
                        return false
                    },
                }, vnode.attrs.loader.loadMore ? 'Load more' : '(loading)')),
            ])),
        ])
    },
}

window.components.collection_search = {
    oninit: function(vnode) {
        vnode.state.sessionDB = new window.models.SessionDB()
        vnode.state.searchEntered = m.stream('')
        vnode.state.searchStart = m.stream('')
        vnode.state.searchStart.map(function(q) {
            vnode.state.loader = new window.models.MultisiteLoader({
                loadFunc: function(session, filters) {
                    if (q)
                        filters.push(['any', '@@', q+':*'])
                    return vnode.state.sessionDB.request(session, 'arvados/v1/collections', {
                        data: {
                            filters: JSON.stringify(filters),
                            count: 'none',
                        },
                    })
                },
                sessionDB: vnode.state.sessionDB,
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
                                className: !vnode.state.loader.pagers[key] ? 'label-default' :
                                    vnode.state.loader.pagers[key].items() ? 'label-success' :
                                    'label-warning',
                            }, key), ' ']
                        }),
                    ' ',
                    m('a[href="/sessions"]', 'Add/remove sites'),
                ]),
            ]),
            m(window.components.collection_table, {
                loader: vnode.state.loader,
            }),
        ])
    },
}
