var jQuery = window.$ = window.jQuery = require('jquery');
var bootstrap = require('bootstrap');
var lte = require('admin-lte');
var m = require('mithril');
var arvados = require('./arvados');
var local = require('./local');

var savedTokens = new local.Dict('tokens');
var _sessions = {};
var resources = [
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
];

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

var GenericCell = {
    view: function(vnode) {
        if (vnode.attrs.field == 'modified_by_user_uuid') {
            var u = (vnode.attrs.session.client.Get('users/'+vnode.attrs.value)() || {})
            return [u.full_name];
        } else if (vnode.attrs.field == 'script_version') {
            return [(vnode.attrs.value || '').slice(0,7)];
        } else if (vnode.attrs.field == 'output') {
            var c = (vnode.attrs.session.client.Get('collections/'+vnode.attrs.item[vnode.attrs.field])() || {})
            return [c.name || c.portable_data_hash];
        } else {
            return [vnode.attrs.value];
        }
    },
};

// m(GenericResourceList(resource), attrs)
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
        vnode.state.coldefs = [];
        vnode.state.session.dd.run(vnode.state.updateColdefs.bind(this, vnode));
    },
    updateColdefs: function(vnode, dd) {
        var schema = dd.resources[vnode.state.resource].methods.get.response['$ref'];
        var props = dd.schemas[schema].properties;
        vnode.state.coldefs =
            ['uuid', 'state', 'hostname', 'ip_address', 'last_ping_at', 'full_name', 'name', 'portable_data_hash', 'script', 'script_version', 'output', 'modified_by_user_uuid'].
            reduce(function(cols, field) {
                console.log(cols);
                if (field in props)
                    cols.push(field);
                return cols;
            }, []);
    },
    view: function(vnode) {
        if (!vnode.state.req()) return m(Loading);
        return m('table.table.table-hover.table-condensed',
                 m('thead',
                   m('tr',
                     vnode.state.coldefs.map(function(field) {
                         return m('th', field);
                     }))),
                 m('tbody',
                   vnode.state.req().items.map(function(item) {
                       return m('tr', {key: item.uuid},
                                vnode.state.coldefs.map(function(field) {
                                    return m('td', m(GenericCell, {
                                        session: vnode.state.session,
                                        item: item,
                                        field: field,
                                        value: item[field],
                                    }));
                                }));
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

var TopNav = {};
TopNav.view = function(vnode) {
    return ;
};

var Head = {
    view: function() {
        return [
            m('link[rel=stylesheet][href=node_modules/bootstrap/dist/css/bootstrap.min.css]'),
            m('link[rel=stylesheet][href=node_modules/font-awesome/css/font-awesome.min.css]'),
            m('link[rel=stylesheet][href=node_modules/ionicons/dist/css/ionicons.min.css]'),
            m('link[rel=stylesheet][href=node_modules/admin-lte/dist/css/AdminLTE.min.css]'),
            m('link[rel=stylesheet][href=node_modules/admin-lte/dist/css/skins/skin-blue.min.css]'),
            m('meta[charset=utf-8]'),
            m('meta[name=viewport][content=width=device-width, initial-scale=1, shrink-to-fit=no]'),
            m('meta[http-equiv=x-ua-compatible][content=ie=edge]'),
            m('style[type=text/css]', 'html, body { min-height: 100%; margin: 0; } .wrapper, .content-wrapper { min-height: 100%; }'),
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
        return m('.wrapper', [
            m('header.main-header',
              m('a.logo[href=/]',
                m('span.logo-mini', 'wb2'),
                m('span.logo-lg', 'wb2')),
              m('nav.navbar.navbar-static-top[role=navigation]',
                m('a.sidebar-toggle[href=#][data-toggle=offcanvas][role=button]',
                  m('span.sr-only', 'Toggle navigation')),
                m('.navbar-custom-menu',
                  m('ul.nav.navbar-nav',
                    m('li.dropdown.notifications-menu',
                      m('a.dropdown-toggle[href=#][data-toggle=dropdown][aria-expanded=false]',
                        m('i.fa.fa-cloud'),
                        m('span.label.label-success',
                          {className: 'label-'+(savedTokens.Get(vnode.attrs.SiteID)?'success':'warning')},
                          vnode.attrs.siteID)),
                      m('ul.dropdown-menu',
                        m('li.header', 'switch site...'),
                        m('li',
                          m('ul.menu',
                            Object.keys(savedTokens.Load()).map(function(siteID) {
                                if (savedTokens.Get(siteID))
                                    return m('li', {key: siteID}, m('a', {
                                        href: '/site/'+siteID+'/'+(vnode.attrs.resource || 'discovery'),
                                        oncreate: m.route.link,
                                    }, siteID));
                                else
                                    return m('li', {key: siteID}, m('a', {
                                        href: getSession(siteID).client.LoginURL(location.href.replace(/([^\/]*\/+[^\/]+[#!?\/]*)/, '$1loginCallback/'+siteID+'/XYZZY/')),
                                    }, 'Log in to '+siteID));
                                return m('a', {
				    oncreate: m.route.link,
				    href: '/site/'+siteID+'/discovery',
				    key: '_site',
			        }, 'about '+siteID);
                            }))),
                        m('li.header',
                          m('i', 'For now, add new sites by editing the location bar.')))))))),
            m('aside.main-sidebar',
              m('section.sidebar',
                m('ul.sidebar-menu',
                  m('li.header', 'resources @ '+vnode.attrs.siteID),
                  resources.map(function(resource) {
                      return m('li', m('a', {
                          oncreate: m.route.link,
                          href: '/site/'+vnode.attrs.siteID+'/'+resource,
                          key: resource,
                      }, resource.replace(/_/g, ' ')));
                  })))),
            m('.content-wrapper',
              m('section.content-header'),
              m('section.content', vnode.attrs, vnode.children)),
        ]);
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

function RouteResolver(layout, component, addAttrs) {
    return {
        render: function(vnode) {
            var attrs = Object.assign({
                key: m.route.get(),
            }, addAttrs || {}, vnode.attrs);
            return m(layout, attrs,
                     m(component, attrs));
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
    resources.map(function(table) {
        routes['/site/:siteID/'+table+'/:uuid'] = RR(Layout, Show, {resource: table});
        routes['/site/:siteID/'+table] = RR(Layout, GenericResourceList(table), {resource: table, filters: []});
    });
    document.body.className = 'skin-blue sidebar-mini';
    m.route(document.body, '/', routes);
    m.mount(document.head, Head);
})();

window.m = m;
