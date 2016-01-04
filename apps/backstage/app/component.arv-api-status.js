module.exports = ArvApiStatusComponent;

var m = require('mithril')
, util = require('app/util')
, DataManagerGraph = require('./component.dmgraph');

function ArvApiStatusComponent(connection) {
    var apistatus = {};
    apistatus.vm = (function() {
        var vm = {};
        vm.connection = connection;
        vm.currentUser = m.prop({});
        vm.dd = connection.discoveryDoc;
        vm.dirty = true;
        vm.init = function() {
            if (vm.dirty)
                vm.refresh();
            vm.dirty = false;
        };
        vm.keepServices = m.prop([]);
        vm.nodes = m.prop([]);
        vm.refresh = function() {
            vm.connection.api(
                'KeepService', 'list', {}).then(vm.keepServices).then(m.redraw);
            vm.connection.api(
                'Node', 'list', {}).then(vm.nodes).then(m.redraw);
            vm.connection.api(
                'User', 'current', {}).then(vm.currentUser).then(m.redraw);
        };
        vm.logout = function() {
            vm.connection.token(undefined);
        };
        vm.ddSummary = function() {
            return !vm.dd() ? {} : {
                apiVersion: vm.dd().version + ' (' + vm.dd().revision + ')',
                sourceVersion: m('a', {
                    href: 'https://dev.arvados.org/projects/arvados/repository/changes?rev=' + vm.dd().source_version.replace(/-.*/,'')
                }, vm.dd().source_version),
                generatedAt: vm.dd().generatedAt,
                websocket: util.choose(vm.connection.webSocket().readyState, {
                    0: m('span.label.label-warning', ['connecting']),
                    1: m('span.label.label-success', ['OK']),
                    2: m('span.label.label-danger', ['closing']),
                    3: m('span.label.label-danger', ['closed']),
                }) || m('span.label.label-danger',
                        {title: ('advertised websocketUrl: ' +
                                 vm.dd().websocketUrl)}, ['none'])
            }
        };
        vm.dmGraphCtrl = new DataManagerGraph.controller({
            connection: connection,
        });
        return vm;
    })();
    apistatus.controller = function() {
        apistatus.vm.init();
    };
    apistatus.view = function() {
        var vm = apistatus.vm;
        var ddSummary = vm.ddSummary();
        return m('.panel.panel-info.arv-bs-api-status', [
            m('.panel-heading', [
                vm.connection.apiPrefix(),
                !vm.dd() ? '' : m('.pull-right', [
                    util.choose(!!vm.connection.token(), {
                        true: [function() {
                            return [vm.currentUser().email,
                                    " ",
                                    m('a.btn.btn-xs.btn-default',
                                      {onclick: vm.logout}, 'Log out')];
                        }],
                        false: [function() {
                            return m('a.btn.btn-xs.btn-primary',
                                     {href: vm.connection.loginLink()}, 'Log in');
                        }]
                    }),
                ]),
            ]),
            m('.panel-body', !vm.dd() ? [vm.connection.state()] : [
                m('.row', [
                    m('.col-md-4',
                      Object.keys(ddSummary).map(function(key) {
                          return m('.row', [
                              m('.col-sm-4.lighten', key),
                              m('.col-sm-8', ddSummary[key]),
                          ]);
                      })),
                    m('.col-md-4', [
                        m('ul', [
                            '' + vm.keepServices().length + ' Keep services',
                            vm.keepServices().map(function(keepService) {
                                return m('li', [
                                    m('span.label.label-default',
                                      keepService.service_type),
                                    ' ',
                                    m('a',
                                      {href: '/show/'+keepService.uuid,
                                       config: m.route}, [
                                           keepService.service_host,
                                           ':',
                                           keepService.service_port,
                                       ]),
                                ]);
                            }),
                        ]),
                    ]),
                    m('.col-md-4', [
                        m('ul', [
                            '' + vm.nodes().length + ' worker nodes',
                            vm.nodes().filter(function(node) {
                                return node.crunch_worker_state != 'down';
                            }).map(function(node) {
                                return m('li', [
                                    m('span.label.label-default', [
                                        node.crunch_worker_state,
                                    ]),
                                    ' ',
                                    m('a', {href: '/show/'+node.uuid,
                                            config: m.route},
                                      node.hostname),
                                    ' ',
                                    m('span.label.label-info', {title: 'time since last ping'}, [
                                        ((new Date() - Date.parse(node.last_ping_at))/1000).toFixed(),
                                        's'
                                    ]),
                                ]);
                            }),
                        ]),
                    ]),
                ]),
                m('.row', 'Collection Job PipelineInstance'.split(' ').map(function(arvModelName) {
                    return m('.col-sm-2', [
                        m('a.btn.btn-xs.btn-default', {
                            style: 'width: 100%',
                            href: '/list/'+vm.connection.apiPrefix()+'/'+arvModelName,
                            config: m.route
                        }, arvModelName+'s'),
                    ]);
                })),
                m('.row', [
                  m('.col-sm-12', [
                      DataManagerGraph.view(vm.dmGraphCtrl)
                  ]),
                ]),
            ]),
        ]);
    };
    return apistatus;
}
