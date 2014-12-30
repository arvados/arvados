module.exports = true;

var m = require('mithril')
, ArvadosConnection = require('arvados/client')
, Layout = require('app/base-layout')
, BackstageLayoutView = require('app/backstage-layout')
, BackstageLoginComponent = require('app/backstage-login')
, ArvApiDirectoryComponent = require('app/component.arv-api-directory')
, ArvIndexComponent = require('app/component.arv-index')
, ArvShowComponent = require('app/component.arv-show');

window.jQuery = require('jquery');
require('bootstrap');

var connections = m.prop('4xphq qr1hi 9tee4 su92l bogus'.split(' ').map(
    function(site) {
        return ArvadosConnection.make(site);
    }));

m.route(document.body, '/', {
    '/login-callback': new BackstageLoginComponent(),
    '/': new Layout(BackstageLayoutView, {
        content: new ArvApiDirectoryComponent(connections)
    }),
    '/list/:connection/:modelName': new Layout(BackstageLayoutView, {
        content: ArvIndexComponent
    }),
    '/show/:uuid': new Layout(BackstageLayoutView, {
        content: ArvShowComponent
    }),
});
