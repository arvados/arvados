var m = require('mithril');
//var arvados = require('./arvados');

var KnownSites = {
    '4xphq': '4xphq.arvadosapi.com',
};

var Sites = {};

var Loading = {
    oninit: function(vnode) {
        m.request({
            method: 'GET',
            url: 'https://'+vnode.attrs.siteID+'.arvadosapi.com/discovery/v1/apis/arvados/v1/rest',
        }).run(function(dd){
            Sites[vnode.attrs.siteID] = {dd: dd};
        });
    },
    view: function() {
        return m('.loading', 'Loading...');
    },
};

var DiscoveryDoc = {
    view: function(vnode) {
        return m('.dd', [
            m('.row', ['site ID: ', vnode.attrs.siteID]),
            m('.row', ['version: ', vnode.attrs.site.dd.source_version]),
            m('.row', ['websocketUrl: ', vnode.attrs.site.dd.websocketUrl]),
            m('.row', ['defaultCollectionReplication: ', vnode.attrs.site.dd.defaultCollectionReplication]),
        ]);
    },
};

var Show = {
    view: function() {
        return m('.show', 'It\'s a collection');
    },
};

var routes = {
    '/': Show,
    '/site/:siteID/discovery': afterLoadingSite(DiscoveryDoc),
};
['collections', 'containers'].map(function(table) {
    routes['/site/:siteID/'+table+'/:uuid'] = afterLoadingSite(Show);
});
m.route(document.body, '/', routes);

function afterLoadingSite(component) {
    return {
        render: function(vnode) {
            var site = Sites[vnode.attrs.siteID];
            return (site
                    ? m(component, Object.assign({}, vnode.attrs, {site: site}))
                    : m(Loading, vnode.attrs));
        },
    };
}
