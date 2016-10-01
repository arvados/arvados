var m = require('mithril');
var arvados = require('./arvados');
var local = require('./local');

var savedTokens = new local.Dict('tokens');
var _sessions = {};

// getSession returns a new or existing session for the API endpoint
// specified by siteID.
function getSession(siteID) {
    var session = _sessions[siteID];
    if (!session) {
        var token = savedTokens.Get(siteID) || savedTokens.Put(siteID, '');
        var client = new arvados.Client(siteID, token, requestFunc);
        session = _sessions[siteID] = {
            client: client,
            dd: client.DiscoveryDoc(),
            websocket: m.prop(),
            token: token,
        };
    }
    return session;
}

function requestFunc(options) {
    if ('headers' in options) {
        var headers = options.headers;
        options.config = function(xhr) {
            headers.map(function(hdr) {
                xhr.setRequestHeader(hdr[0], hdr[1]);
            });
            return xhr;
        }
        delete options.headers;
    }
    return m.request(options);
}

// remove me
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

var FormRow = {
    view: function(vnode) {
        return m('.form-group.row',
                 m('label.col-sm-3.col-form-label', vnode.attrs.label),
                 m('.col-sm-9',
                   m('p.form-control-static', vnode.attrs.value)));
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
        return m('form', [
            m(FormRow, {type: 'static', label: 'site ID', value: vnode.attrs.siteID}),
            m(FormRow, {type: 'static', label: 'version', value: dd().source_version}),
            m(FormRow, {type: 'static', label: 'websocketUrl', value: dd().websocketUrl}),
            m(FormRow, {type: 'static', label: 'defaultCollectionReplication', value: dd().defaultCollectionReplication}),
            vnode.state.current_user() ? m(FormRow, {
                type: 'static',
                label: 'current user',
                value: [vnode.state.current_user().full_name,
                        ' (', vnode.state.current_user().username,
                        ', ', vnode.state.current_user().email,
                        ')'],
            }) : [],
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

// m(GenericResourceList, attrs, children...)
//
// resource: arvados resource path, e.g., 'collections'
// attrs.filters: an array of arvados filters, or a stream that returns one
var GenericResourceList = function(resource) { return {
    oninit: function(vnode) {
        vnode.state.resource = resource;
        vnode.state.session = getSession(vnode.attrs.siteID);
        vnode.state.req =
            vnode.state.session.client.Get(vnode.state.resource, {
                filters: (vnode.attrs.filters instanceof Array) ? vnode.attrs.filters : vnode.attrs.filters(),
            }).
            catch(function() { return {items: []} });
    },
    view: function(vnode) {
        return !vnode.state.req() ? m(Loading) : m('table.table.table-hover.table-sm',
                 m('tbody',
                   vnode.state.req().items.map(function(item) {
                       var user = vnode.state.session.client.Get('users/'+item.modified_by_user_uuid)() || {};
                       return m('tr',
                                m('td', item.full_name || item.name || item.hostname || item.script),
                                m('td', item.email || item.script_version),
                                m('td', user.full_name),
                                m('td', item.uuid));
                   })));
    },
}};

var bsDropdown = {
    oninit: function(vnode) {
        vnode.state.toggle = function(e) {
            vnode.state.open = !vnode.state.open;
            return false;
        };
    },
    view: function(vnode) {
        return m('.dropdown', {className: vnode.state.open ? 'open' : ''}, [
            m('a.btn.btn-secondary.dropdown-toggle', {
                onclick: vnode.state.toggle,
            }, vnode.attrs.label),
            m('.dropdown-menu',
              {className: vnode.attrs.align == 'right' ? 'dropdown-menu-right' : ''},
              vnode.attrs.items.map(function(item) {
                  item.attrs.className = 'dropdown-item '+item.attrs.className;
                  return item;
              })),
        ]);
    },
};

var TopNav = {
    view: function(vnode) {
        return m('nav.navbar.navbar-light[style=background-color:#e3f2fd]',
                 m('.pull-xs-right',
                   m(bsDropdown, {
                       label: 'Log in...',
                       align: 'right',
                       items: Object.keys(savedTokens.Load()).map(function(siteID) {
                           return m('a', {
                               key: siteID,
                               href: getSession(siteID).client.LoginURL(location.href.replace(/([^\/]*\/+[^\/]+[#!?\/]*)/, '$1loginCallback/'+siteID+'/XYZZY/')),
                           }, siteID);
                       }),
                   })));
    },
};

var Head = {
    view: function() {
        return [
            m('link[rel=stylesheet][href=https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0-alpha.4/css/bootstrap.min.css][integrity=sha384-2hfp1SzUoho7/TsGGGDaFdsuuDL0LX2hnUp6VkX3CUQ2K4K+xjboZdsXyp4oUHZj][crossorigin=anonymous]'),
            m('meta[charset=utf-8]'),
            m('meta[name=viewport][content=width=device-width, initial-scale=1, shrink-to-fit=no]'),
            m('meta[http-equiv=x-ua-compatible][content=ie=edge]'),
        ];
    },
};

var Layout = {
    oninit: function(vnode) {
        // TODO: (here, or in a separate page wrapper?) build map of
        // known/logged-in sites, start getting discovery docs if
        // needed
    },
    view: function(vnode) {
        return [
            m(TopNav),
            m('.container-fluid', vnode.attrs, vnode.children),
        ];
    },
};

var TryLogin = {
    view: function(vnode) {
        var token;
        if (token = location.href.match(/(\?api_token=([^\?&]+))/)) {
            location = location.href.replace(token[1], '').replace('XYZZY', token[2]);
        } else if (token = vnode.attrs.token) {
            savedTokens.Put(vnode.attrs.siteID, token);
            m.route.set('/'+vnode.attrs.next);
        } else {
            m.route.set('/site/4xphq/discovery');
        }
    },
};

function RouteResolver(layout, component, attrs) {
    return {
        render: function(vnode) {
            return m(layout, m(component,
                               Object.assign({
                                   key: m.route.get(),
                               }, attrs || {}, vnode.attrs)));
        },
    };
}

(function SetupRouting() {
    var RR = RouteResolver;
    var routes = {
        '/': TryLogin,
        '/site/:siteID/discovery': RR(Layout, DiscoveryDoc),
        '/loginCallback/:siteID/:token/:next...': TryLogin,
    };
    [
        'api_clients',
        'authorized_keys',
        'collections',
        'container_requests',
        'containers',
        'groups',
        'humans',
        'jobs',
        'job_tasks',
        'nodes',
        'repositories',
        'specimens',
        'users',
        'virtual_machines',
    ].map(function(table) {
        routes['/site/:siteID/'+table+'/:uuid'] = RR(Layout, Show);
        routes['/site/:siteID/'+table] = RR(Layout, GenericResourceList(table), {filters: []});
    });
    m.route(document.body, '/', routes);
    m.mount(document.head, Head);
})();

window.m = m;
