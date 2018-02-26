// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

$(document).on('ready', function() {
    var db = new SessionDB();
    db.checkForNewToken();
    db.fillMissingUUIDs();
    db.autoLoadRemoteHosts();
});

window.SessionsTable = {
    oninit: function(vnode) {
        vnode.state.db = new SessionDB();
        vnode.state.db.autoRedirectToHomeCluster('/sessions');
        vnode.state.db.migrateNonFederatedSessions();
        vnode.state.hostToAdd = m.stream('');
        vnode.state.error = m.stream();
        vnode.state.checking = m.stream();
    },
    view: function(vnode) {
        var db = vnode.state.db;
        var sessions = db.loadAll();
        return m('.container', [
            m('p', [
                'You can log in to multiple Arvados sites here, then use the ',
                m('a[href="/search"]', 'multi-site search'),
                ' page to search collections and projects on all sites at once.'
            ]),
            m('table.table.table-condensed.table-hover', [
                m('thead', m('tr', [
                    m('th', 'status'),
                    m('th', 'cluster ID'),
                    m('th', 'username'),
                    m('th', 'email'),
                    m('th', 'actions'),
                    m('th')
                ])),
                m('tbody', [
                    Object.keys(sessions).map(function(uuidPrefix) {
                        var session = sessions[uuidPrefix];
                        return m('tr', [
                            session.token && session.user ? [
                                m('td', session.user.is_active ?
                                    m('span.label.label-success', 'logged in') :
                                    m('span.label.label-warning', 'inactive')),
                                m('td', {title: session.baseURL}, uuidPrefix),
                                m('td', session.user.username),
                                m('td', session.user.email),
                                m('td', session.isFromRails ? null : m('button.btn.btn-xs.btn-default', {
                                    uuidPrefix: uuidPrefix,
                                    onclick: m.withAttr('uuidPrefix', db.logout),
                                }, session.listedHost ? 'Disable ':'Log out ', m('span.glyphicon.glyphicon-log-out')))
                            ] : [
                                m('td', m('span.label.label-default', 'logged out')),
                                m('td', {title: session.baseURL}, uuidPrefix),
                                m('td'),
                                m('td'),
                                m('td', m('a.btn.btn-xs.btn-primary', {
                                    uuidPrefix: uuidPrefix,
                                    onclick: db.login.bind(db, session.baseURL),
                                }, session.listedHost ? 'Enable ':'Log in ', m('span.glyphicon.glyphicon-log-in')))
                            ],
                            m('td', (session.isFromRails || session.listedHost) ? null :
                                m('button.btn.btn-xs.btn-default', {
                                    uuidPrefix: uuidPrefix,
                                    onclick: m.withAttr('uuidPrefix', db.trash),
                                }, 'Remove ', m('span.glyphicon.glyphicon-trash'))
                            ),
                        ])
                    }),
                ]),
            ]),
            m('.row', m('.col-md-6', [
                m('form', {
                    onsubmit: function() {
                        vnode.state.error(null)
                        vnode.state.checking(true)
                        db.findAPI(vnode.state.hostToAdd())
                            .then(db.login)
                            .catch(function() {
                                vnode.state.error(true)
                            })
                            .then(vnode.state.checking.bind(null, null))
                        return false
                    },
                }, [
                    m('p', [
                        'To add a remote Arvados site, paste the remote site\'s host here (see "ARVADOS_API_HOST" on the "current token" page).',
                    ]),
                    m('.input-group', { className: vnode.state.error() && 'has-error' }, [
                        m('input.form-control[type=text][name=apiHost][placeholder="zzzzz.arvadosapi.com"]', {
                            oninput: m.withAttr('value', vnode.state.hostToAdd),
                        }),
                        m('.input-group-btn', [
                            m('input.btn.btn-primary[type=submit][value="Log in"]', {
                                disabled: !vnode.state.hostToAdd(),
                            }),
                        ]),
                    ]),
                ]),
                m('p'),
                vnode.state.error() && m('p.alert.alert-danger', 'Request failed. Make sure this is a working API server address.'),
                vnode.state.checking() && m('p.alert.alert-info', 'Checking...'),
            ])),
        ])
    },
}
