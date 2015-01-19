module.exports = BackstageLayout;

var m = require('mithril');
var _ = require('lodash');
var Layout = require('./base-layout');

function BackstageLayout(innerModules) {
    return _.extend(this, {
        controller: BackstageLayout.controller.bind(this, innerModules),
        view: BackstageLayout.view,
    });
}
BackstageLayout.prototype = new Layout();
BackstageLayout.controller = Layout.controller;
BackstageLayout.view = function view(ctrl) {
    return [
        m('.navbar.navbar-default', {role: 'navigation'}, [
            m('.container-fluid', [
                m('.navbar-header', [
                    m('button.navbar-toggle.collapsed',
                      {'data-toggle': 'collapse', 'data-target': '#navbar'},
                      [0,0,0].map(function() {
                          return m('span.icon-bar');
                      })),
                    m("a.navbar-brand[href='/']", {config:m.route},
                      'Arvados::Backstage'),
                ]),
                m('#navbar.navbar-collapse.collapse', [
                    m('ul.nav.navbar-nav', [
                        m('li', [
                            m("a[href='/']", {config:m.route},
                              'Dashboard'),
                        ]),
                    ]),
                    m('p.navbar-text', [siteBreadcrumb()]),
                ]),
            ]),
        ]),
        m('.container-fluid', ctrl.views.content()),
    ];
    function siteBreadcrumb() {
        var txt;
        if (txt = m.route.param('connection'))
            return txt;
        if ((txt = m.route.param('uuid')) && txt.substr(5,1)=='-')
            return txt.substr(0,5);
        return '';
    }
}
