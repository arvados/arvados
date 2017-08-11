// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.models = window.models || {}
window.models.SessionDB = function() {
    var db = this
    Object.assign(db, {
        loadFromLocalStorage: function() {
            try {
                return JSON.parse(window.localStorage.getItem('sessions')) || {}
            } catch(e) {}
            return {}
        },
        loadAll: function() {
            var all = db.loadFromLocalStorage()
            if (window.defaultSession) {
                window.defaultSession.isFromRails = true
                all[window.defaultSession.user.uuid.slice(0, 5)] = window.defaultSession
            }
            return all
        },
        loadActive: function() {
            var sessions = db.loadAll()
            Object.keys(sessions).forEach(function(key) {
                if (!sessions[key].token)
                    delete sessions[key]
            })
            return sessions
        },
        save: function(k, v) {
            var sessions = db.loadAll()
            sessions[k] = v
            Object.keys(sessions).forEach(function(key) {
                if (sessions[key].isFromRails)
                    delete sessions[key]
            })
            window.localStorage.setItem('sessions', JSON.stringify(sessions))
        },
        trash: function(k) {
            var sessions = db.loadAll()
            delete sessions[k]
            window.localStorage.setItem('sessions', JSON.stringify(sessions))
        },
        login: function(host) {
            // Initiate login procedure with given API host (which can
            // optionally include scheme://).
            //
            // Any page that has a button that invokes login() must
            // also call checkForNewToken() on (at least) its first
            // render. Otherwise, the login procedure can't be
            // completed.
            var baseURL = host
            if (baseURL.indexOf('://') < 0)
                baseURL = 'https://' + baseURL
            if (!baseURL.endsWith('/'))
                baseURL = baseURL + '/'
            document.location = baseURL + 'login?return_to=' + encodeURIComponent(document.location.href.replace(/\?.*/, '')+'?baseURL='+encodeURIComponent(baseURL))
            return false
        },
        logout: function(k) {
            // Forget the token, but leave the other info in the db so
            // the user can log in again without providing the login
            // host again.
            var sessions = db.loadAll()
            delete sessions[k].token
            db.save(k, sessions[k])
        },
        checkForNewToken: function() {
            // If there's a token and baseURL in the location bar (i.e.,
            // we just landed here after a successful login), save it and
            // scrub the location bar.
            if (!document.location.search.startsWith('?'))
                return
            var params = {}
            document.location.search.slice(1).split('&').map(function(kv) {
                var e = kv.indexOf('=')
                if (e < 0)
                    return
                params[decodeURIComponent(kv.slice(0, e))] = decodeURIComponent(kv.slice(e+1))
            })
            if (!params.baseURL || !params.api_token)
                // Have a query string, but it's not a login callback.
                return
            params.token = params.api_token
            delete params.api_token
            db.save(params.baseURL, params)
            history.replaceState({}, '', document.location.origin + document.location.pathname)
        },
        fillMissingUUIDs: function() {
            var sessions = db.loadAll()
            Object.keys(sessions).map(function(key) {
                if (key.indexOf('://') < 0)
                    return
                // key is the baseURL placeholder. We need to get our user
                // record to find out the cluster's real uuid prefix.
                var session = sessions[key]
                m.request(session.baseURL+'arvados/v1/users/current', {
                    headers: {
                        authorization: 'OAuth2 '+session.token,
                    },
                }).then(function(user) {
                    session.user = user
                    db.save(user.uuid.slice(0, 5), session)
                    db.trash(key)
                })
            })
            // m.request(session.baseURL + 'discovery/v1/apis/arvados/v1/rest').then(function(dd) {})
        },
        request: function(session, path, opts) {
            opts = opts || {}
            opts.headers = opts.headers || {}
            opts.headers.authorization = 'OAuth2 '+ session.token
            return m.request(session.baseURL + path, opts)
        },
    })
}
