var m = require('mithril');
//var arvados = require('./arvados');

var KnownSites = {
    '4xphq': '4xphq.arvadosapi.com',
};

var ddLoaded = {};
function ddLoad(siteID) {
    var dd = ddLoaded[siteID];
    if (!dd) dd = ddLoaded[siteID] = m.prop();
    else if (dd()) return dd;
    m.request({
        method: 'GET',
        url: 'https://'+siteID+'.arvadosapi.com/discovery/v1/apis/arvados/v1/rest',
    }).run(dd);
    return dd;
}

var Loading = {
    oninit: function(vnode) {
    },
    view: function() {
        return m('.loading', 'Loading...');
    },
};

var DiscoveryDoc = {
    view: function(vnode) {
        var dd = ddLoad(vnode.attrs.siteID);
        if (!dd()) return m(Loading);
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
        var dd = ddLoad(vnode.attrs.siteID);
        if (!dd()) return m(Loading);
        return m('.show', 'It\'s a collection from ', vnode.attrs.siteID);
    },
};

var Layout = {
    RouteResolver: function(component) {
        return {
            render: function(vnode) {
                return m(Layout, m(component, vnode.attrs));
            },
        };
    },
    view: function(vnode) {
        return m('.layout', vnode.children);
    },
};

var routes = {
    '/': Show,
    '/site/:siteID/discovery': Layout.RouteResolver(DiscoveryDoc),
};
['collections', 'containers'].map(function(table) {
    routes['/site/:siteID/'+table+'/:uuid'] = Layout.RouteResolver(Show);
});
m.route(document.body, '/', routes);
