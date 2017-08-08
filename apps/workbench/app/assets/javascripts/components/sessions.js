// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

$(document).on('ready', function() {
    var db = new window.models.SessionDB()
    db.checkForNewToken()
    db.fillMissingUUIDs()
})

window.components = window.components || {}
window.components.sessions = {
    oninit: function(vnode) {
        vnode.state.db = new window.models.SessionDB()
        vnode.state.hostToAdd = m.stream('')
    },
    view: function(vnode) {
        var db = vnode.state.db
        var sessions = db.loadAll()
        return m('container', [
            m('table.table.table-condensed.table-hover', m('tbody', [
                Object.keys(sessions).map(function(uuidPrefix) {
                    var session = sessions[uuidPrefix]
                    return m('tr', [
                        session.token && session.user ? [
                            m('td', m('a.btn.btn-xs.btn-default', {
                                uuidPrefix: uuidPrefix,
                                onclick: m.withAttr('uuidPrefix', db.logout),
                            }, 'log out')),
                            m('td', m('span.label.label-info', 'logged in')),
                            m('td', {title: session.baseURL}, uuidPrefix),
                            m('td', session.user.username),
                            m('td', session.user.email),
                        ] : [
                            m('td', m('a.btn.btn-xs.btn-info', {
                                uuidPrefix: uuidPrefix,
                                onclick: m.withAttr('uuidPrefix', db.login),
                            }, 'log in')),
                            m('td', 'span.label.label-default', 'logged out'),
                            m('td', {title: session.baseURL}, uuidPrefix),
                            m('td'),
                            m('td'),
                        ],
                        m('td', m('a.glyphicon.glyphicon-trash', {
                            uuidPrefix: uuidPrefix,
                            onclick: m.withAttr('uuidPrefix', db.trash),
                        })),
                    ])
                }),
            ])),
            m('.row', m('.col-md-6', [
                m('form', {
                    onsubmit: function() {
                        db.login(vnode.state.hostToAdd())
                        return false
                    },
                }, [
                    m('.input-group', [
                        m('input.form-control[type=text][name=apiHost][placeholder="API host"]', {
                            oninput: m.withAttr('value', vnode.state.hostToAdd),
                        }),
                        m('.input-group-btn', [
                            m('input.btn.btn-primary[type=submit][value="Log in"]', {
                                disabled: !vnode.state.hostToAdd(),
                            }),
                        ]),
                    ]),
                ]),
            ])),
        ])
    },
}
