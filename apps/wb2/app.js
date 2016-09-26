var m = require('mithril');
var arvados = require('./arvados');

var _sessions = {};

// getSession returns a new or existing session for the API endpoint
// specified by siteID.
function getSession(siteID) {
    var session = _sessions[siteID];
    if (!session) {
        var client = new arvados.Client(siteID);
        session = _sessions[siteID] = {
            client: client,
            dd: m.request({
                method: 'GET',
                url: client.DiscoveryURL(),
            }),
            websocket: m.prop(),
            token: loadToken(siteID),
        };
    }
    return session;
}

function ArvadosRequest(session, method, url) {
    return session.dd.run(function() {
        return m.request({
            method: method,
            url: session.dd().baseUrl + url,
            config: function(xhr) {
                xhr.setRequestHeader('Authorization', 'OAuth2 '+session.token);
                return xhr;
            },
        });
    });
}

function saveToken(siteID, token) {
    var tokens = {};
    try {
        tokens = JSON.parse(window.localStorage.tokens);
    } catch(e) {}
    tokens[siteID] = token;
    window.localStorage.tokens = JSON.stringify(tokens);
    getSession(siteID).token = token;
}

function loadToken(siteID) {
    try {
        return JSON.parse(window.localStorage.tokens)[siteID];
    } catch(e) {
        return undefined;
    }
}

// getDiscoveryDoc returns a stream resolving to the discovery
// document for the Arvados API endpoint specified by siteID.
function getDiscoveryDoc(siteID) {
    return getSession(siteID).dd;
}

var Loading = {
    view: function() {
        return m('.loading', 'Loading...');
    },
};

var ErrorTODO = {
    view: function(vnode) {
        return m('.errorTODO', 'Error loading: ', vnode.children)
    },
};

var DiscoveryDoc = {
    oninit: function(vnode) {
        vnode.state.dd = getDiscoveryDoc(vnode.attrs.siteID);
        var session = getSession(vnode.attrs.siteID);
        if (session.token)
            vnode.state.current_user = ArvadosRequest(session, 'GET', 'users/current');
        else
            vnode.state.current_user = m.prop();
    },
    view: function(vnode) {
        var dd = vnode.state.dd;
        if (dd.error()) return m(ErrorTODO, dd.error());
        else if (!dd()) return m(Loading);
        return m('.dd', [
            m('.row', ['site ID: ', vnode.attrs.siteID]),
            m('.row', ['version: ', dd().source_version]),
            m('.row', ['websocketUrl: ', dd().websocketUrl]),
            m('.row', ['defaultCollectionReplication: ', dd().defaultCollectionReplication]),
            vnode.state.current_user ? m('.row', [
                'current user: ',
                vnode.state.current_user().full_name,
                ' (', vnode.state.current_user().username,
                ', ', vnode.state.current_user().email,
                ')',
            ]) : [],
        ]);
    },
};

var Show = {
    view: function(vnode) {
        var dd = getDiscoveryDoc(vnode.attrs.siteID);
        if (dd.error()) return m(ErrorTODO, dd.error());
        else if (!dd()) return m(Loading);
        return m('.show', 'It\'s a collection from ', vnode.attrs.siteID);
    },
};

var TopNav = {
    view: function(vnode) {
        return Object.keys(_sessions).map(function(siteID) {
            return [m('a', {
                href: getSession(siteID).client.LoginURL(location.href.replace(/([^\/]*\/+[^\/]+[#!?\/]*)/, '$1loginCallback/'+siteID+'/XYZZY/')),
            }, 'Login:', siteID), m.trust(' &bull; ')];
        });
    },
};

var Layout = {
    view: function(vnode) {
        return m('.layout', m(TopNav), vnode.children);
    },
};

var TryLogin = {
    view: function(vnode) {
        var token;
        if (token = location.href.match(/(\?api_token=([^\?&]+))/)) {
            location = location.href.replace(token[1], '').replace('XYZZY', token[2]);
        } else if (token = vnode.attrs.token) {
            saveToken(vnode.attrs.siteID, token);
            m.route.set('/'+vnode.attrs.next);
        } else {
            m.route.set('/site/4xphq/discovery');
        }
    },
};

function RouteResolver(layout, component, withKey) {
    return {
        render: function(vnode) {
            return m(layout, m(component,
                               Object.assign({
                                   key: withKey + ':' + vnode.attrs[withKey],
                               }, vnode.attrs)));
        },
    };
}

(function SetupRouting() {
    var RR = RouteResolver;
    var routes = {
        '/': TryLogin,
        '/site/:siteID/discovery': RR(Layout, DiscoveryDoc, 'siteID'),
        '/loginCallback/:siteID/:token/:next...': TryLogin,
    };
    ['collections', 'containers'].map(function(table) {
        routes['/site/:siteID/'+table+'/:uuid'] = RR(Layout, Show, 'uuid');
    });
    m.route(document.body, '/', routes);
})();
