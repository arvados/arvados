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
        };
    }
    return session;
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

var Layout = {
    RouteResolver: function(component, withKey) {
        return {
            render: function(vnode) {
                return m(Layout, m(component,
                                   Object.assign({
                                       key: vnode.attrs[withKey],
                                   }, vnode.attrs)));
            },
        };
    },
    view: function(vnode) {
        return m('.layout', vnode.children);
    },
};

var routes = {
    '/': Show,
    '/site/:siteID/discovery': Layout.RouteResolver(DiscoveryDoc, 'siteID'),
};
['collections', 'containers'].map(function(table) {
    routes['/site/:siteID/'+table+'/:uuid'] = Layout.RouteResolver(Show, 'uuid');
});
m.route(document.body, '/', routes);
