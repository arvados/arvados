// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.SessionDB = function() {
    var db = this;
    Object.assign(db, {
        discoveryCache: {},
        tokenUUIDCache: null,
        loadFromLocalStorage: function() {
            try {
                return JSON.parse(window.localStorage.getItem('sessions')) || {};
            } catch(e) {}
            return {};
        },
        loadAll: function() {
            var all = db.loadFromLocalStorage();
            if (window.defaultSession) {
                window.defaultSession.isFromRails = true;
                all[window.defaultSession.user.uuid.slice(0, 5)] = window.defaultSession;
            }
            return all;
        },
        loadActive: function() {
            var sessions = db.loadAll();
            Object.keys(sessions).forEach(function(key) {
                if (!sessions[key].token || (sessions[key].user && !sessions[key].user.is_active)) {
                    delete sessions[key];
                }
            });
            return sessions;
        },
        loadLocal: function() {
            var sessions = db.loadActive();
            var s = false;
            Object.keys(sessions).forEach(function(key) {
                if (sessions[key].isFromRails) {
                    s = sessions[key];
                    return;
                }
            });
            return s;
        },
        save: function(k, v) {
            var sessions = db.loadAll();
            sessions[k] = v;
            Object.keys(sessions).forEach(function(key) {
                if (sessions[key].isFromRails) {
                    delete sessions[key];
                }
            });
            window.localStorage.setItem('sessions', JSON.stringify(sessions));
        },
        trash: function(k) {
            var sessions = db.loadAll();
            delete sessions[k];
            window.localStorage.setItem('sessions', JSON.stringify(sessions));
        },
        findAPI: function(url) {
            // Given a Workbench or API host or URL, return a promise
            // for the corresponding API server's base URL.  Typical
            // use:
            // sessionDB.findAPI('https://workbench.example/foo').then(sessionDB.login)
            if (url.length === 5 && url.indexOf('.') < 0) {
                url += '.arvadosapi.com';
            }
            if (url.indexOf('://') < 0) {
                url = 'https://' + url;
            }
            url = new URL(url);
            return m.request(url.origin + '/discovery/v1/apis/arvados/v1/rest').then(function() {
                return url.origin + '/';
            }).catch(function(err) {
                // If url is a Workbench site (and isn't too old),
                // /status.json will tell us its API host.
                return m.request(url.origin + '/status.json').then(function(resp) {
                    if (!resp.apiBaseURL) {
                        throw 'no apiBaseURL in status response';
                    }
                    return resp.apiBaseURL;
                });
            });
        },
        login: function(baseURL, fallbackLogin) {
            // Initiate login procedure with given API base URL (e.g.,
            // "http://api.example/").
            //
            // Any page that has a button that invokes login() must
            // also call checkForNewToken() on (at least) its first
            // render. Otherwise, the login procedure can't be
            // completed.
            if (fallbackLogin === undefined) {
                fallbackLogin = true;
            }
            var session = db.loadLocal();
            var apiHostname = new URL(session.baseURL).hostname;
            db.discoveryDoc(session).map(function(localDD) {
                var uuidPrefix = localDD.uuidPrefix;
                db.discoveryDoc({baseURL: baseURL}).map(function(dd) {
                    if (uuidPrefix in dd.remoteHosts ||
                        (dd.remoteHostsViaDNS && apiHostname.endsWith('.arvadosapi.com'))) {
                        // Federated identity login via salted token
                        db.saltedToken(dd.uuidPrefix).then(function(token) {
                            m.request(baseURL+'arvados/v1/users/current', {
                                headers: {
                                    authorization: 'Bearer '+token
                                }
                            }).then(function(user) {
                                // Federated login successful.
                                var remoteSession = {
                                    user: user,
                                    baseURL: baseURL,
                                    token: token,
                                    listedHost: (dd.uuidPrefix in localDD.remoteHosts)
                                };
                                db.save(dd.uuidPrefix, remoteSession);
                            }).catch(function(e) {
                                if (dd.uuidPrefix in localDD.remoteHosts) {
                                    // If the remote system is configured to allow federated
                                    // logins from this cluster, but rejected the salted
                                    // token, save as a logged out session anyways.
                                    var remoteSession = {
                                        baseURL: baseURL,
                                        listedHost: true
                                    };
                                    db.save(dd.uuidPrefix, remoteSession);
                                } else if (fallbackLogin) {
                                    // Remote cluster not listed as a remote host and rejecting
                                    // the salted token, try classic login.
                                    db.loginClassic(baseURL);
                                }
                            });
                        });
                    } else if (fallbackLogin) {
                        // Classic login will be used when the remote system doesn't list this
                        // cluster as part of the federation.
                        db.loginClassic(baseURL);
                    }
                });
            });
            return false;
        },
        loginClassic: function(baseURL) {
            document.location = baseURL + 'login?return_to=' + encodeURIComponent(document.location.href.replace(/\?.*/, '')+'?baseURL='+encodeURIComponent(baseURL));
        },
        logout: function(k) {
            // Forget the token, but leave the other info in the db so
            // the user can log in again without providing the login
            // host again.
            var sessions = db.loadAll();
            delete sessions[k].token;
            db.save(k, sessions[k]);
        },
        saltedToken: function(uuid_prefix) {
            // Takes a cluster UUID prefix and returns a salted token to allow
            // log into said cluster using federated identity.
            var session = db.loadLocal();
            return db.tokenUUID().then(function(token_uuid) {
                var shaObj = new jsSHA("SHA-1", "TEXT");
                var secret = session.token;
                if (session.token.startsWith("v2/")) {
                    secret = session.token.split("/")[2];
                }
                shaObj.setHMACKey(secret, "TEXT");
                shaObj.update(uuid_prefix);
                var hmac = shaObj.getHMAC("HEX");
                return 'v2/' + token_uuid + '/' + hmac;
            });
        },
        checkForNewToken: function() {
            // If there's a token and baseURL in the location bar (i.e.,
            // we just landed here after a successful login), save it and
            // scrub the location bar.
            if (document.location.search[0] != '?') { return; }
            var params = {};
            document.location.search.slice(1).split('&').forEach(function(kv) {
                var e = kv.indexOf('=');
                if (e < 0) {
                    return;
                }
                params[decodeURIComponent(kv.slice(0, e))] = decodeURIComponent(kv.slice(e+1));
            });
            if (!params.baseURL || !params.api_token) {
                // Have a query string, but it's not a login callback.
                return;
            }
            params.token = params.api_token;
            delete params.api_token;
            db.save(params.baseURL, params);
            history.replaceState({}, '', document.location.origin + document.location.pathname);
        },
        fillMissingUUIDs: function() {
            var sessions = db.loadAll();
            Object.keys(sessions).forEach(function(key) {
                if (key.indexOf('://') < 0) {
                    return;
                }
                // key is the baseURL placeholder. We need to get our user
                // record to find out the cluster's real uuid prefix.
                var session = sessions[key];
                m.request(session.baseURL+'arvados/v1/users/current', {
                    headers: {
                        authorization: 'OAuth2 '+session.token
                    }
                }).then(function(user) {
                    session.user = user;
                    db.save(user.owner_uuid.slice(0, 5), session);
                    db.trash(key);
                });
            });
        },
        // Return the Workbench base URL advertised by the session's
        // API server, or a reasonable guess, or (if neither strategy
        // works out) null.
        workbenchBaseURL: function(session) {
            var dd = db.discoveryDoc(session)();
            if (!dd) {
                // Don't fall back to guessing until we receive the discovery doc
                return null;
            }
            if (dd.workbenchUrl) {
                return dd.workbenchUrl;
            }
            // Guess workbench.{apihostport} is a Workbench... unless
            // the host part of apihostport is an IPv4 or [IPv6]
            // address.
            if (!session.baseURL.match('://(\\[|\\d+\\.\\d+\\.\\d+\\.\\d+[:/])')) {
                var wbUrl = session.baseURL.replace('://', '://workbench.');
                // Remove the trailing slash, if it's there.
                return wbUrl.slice(-1) === '/' ? wbUrl.slice(0, -1) : wbUrl;
            }
            return null;
        },
        // Return a m.stream that will get fulfilled with the
        // discovery doc from a session's API server.
        discoveryDoc: function(session) {
            var cache = db.discoveryCache[session.baseURL];
            if (!cache && session) {
                db.discoveryCache[session.baseURL] = cache = m.stream();
                var baseURL = session.baseURL;
                if (baseURL[baseURL.length - 1] !== '/') {
                    baseURL += '/';
                }
                m.request(baseURL+'discovery/v1/apis/arvados/v1/rest')
                    .then(function (dd) {
                        // Just in case we're talking with an old API server.
                        dd.remoteHosts = dd.remoteHosts || {};
                        if (dd.remoteHostsViaDNS === undefined) {
                            dd.remoteHostsViaDNS = false;
                        }
                        return dd;
                    })
                    .then(cache);
            }
            return cache;
        },
        // Return a promise with the local session token's UUID from the API server.
        tokenUUID: function() {
            var cache = db.tokenUUIDCache;
            if (!cache) {
                var session = db.loadLocal();
                if (session.token.startsWith("v2/")) {
                    var uuid = session.token.split("/")[1]
                    db.tokenUUIDCache = uuid;
                    return new Promise(function(resolve, reject) {
                        resolve(uuid);
                    });
                }
                return db.request(session, 'arvados/v1/api_client_authorizations', {
                    data: {
                        filters: JSON.stringify([['api_token', '=', session.token]])
                    }
                }).then(function(resp) {
                    var uuid = resp.items[0].uuid;
                    db.tokenUUIDCache = uuid;
                    return uuid;
                });
            } else {
                return new Promise(function(resolve, reject) {
                    resolve(cache);
                });
            }
        },
        request: function(session, path, opts) {
            opts = opts || {};
            opts.headers = opts.headers || {};
            opts.headers.authorization = 'OAuth2 '+ session.token;
            return m.request(session.baseURL + path, opts);
        },
        // Check non-federated remote active sessions if they should be migrated to
        // a salted token.
        migrateNonFederatedSessions: function() {
            var sessions = db.loadActive();
            Object.keys(sessions).forEach(function(uuidPrefix) {
                session = sessions[uuidPrefix];
                if (!session.isFromRails && session.token) {
                    db.saltedToken(uuidPrefix).then(function(saltedToken) {
                        if (session.token != saltedToken) {
                            // Only try the federated login
                            db.login(session.baseURL, false);
                        }
                    });
                }
            });
        },
        // If remoteHosts is populated on the local API discovery doc, try to
        // add any listed missing session.
        autoLoadRemoteHosts: function() {
            var sessions = db.loadAll();
            var doc = db.discoveryDoc(db.loadLocal());
            if (doc === undefined) { return; }
            doc.map(function(d) {
                Object.keys(d.remoteHosts).forEach(function(uuidPrefix) {
                    if (!(sessions[uuidPrefix])) {
                        db.findAPI(d.remoteHosts[uuidPrefix]).then(function(baseURL) {
                            db.login(baseURL, false);
                        });
                    }
                });
            });
        },
        // If the current logged in account is from a remote federated cluster,
        // redirect the user to their home cluster's workbench.
        // This is meant to avoid confusion when the user clicks through a search
        // result on the home cluster's multi site search page, landing on the
        // remote workbench and later trying to do another search by just clicking
        // on the multi site search button instead of going back with the browser.
        autoRedirectToHomeCluster: function(path) {
            path = path || '/';
            var session = db.loadLocal();
            var userUUIDPrefix = session.user.uuid.slice(0, 5);
            // If the current user is local to the cluster, do nothing.
            if (userUUIDPrefix === session.user.owner_uuid.slice(0, 5)) {
                return;
            }
            db.discoveryDoc(session).map(function (d) {
                // Guess the remote host from the local discovery doc settings
                var rHost = null;
                if (d.remoteHosts[userUUIDPrefix]) {
                    rHost = d.remoteHosts[userUUIDPrefix];
                } else if (d.remoteHostsViaDNS) {
                    rHost = userUUIDPrefix + '.arvadosapi.com';
                } else {
                    // This should not happen: having remote user whose uuid prefix
                    // isn't listed on remoteHosts and dns mechanism is deactivated
                    return;
                }
                // Get the remote cluster workbench url & redirect there.
                db.findAPI(rHost).then(function (apiUrl) {
                    db.discoveryDoc({baseURL: apiUrl}).map(function (d) {
                        document.location = d.workbenchUrl + path;
                    });
                });
            });
        }
    });
};
