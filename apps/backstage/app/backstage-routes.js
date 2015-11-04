module.exports = true;

var m = require('mithril')
, ArvadosConnection = require('arvados/client')
, Layout = require('./base-layout')
, BackstageLayout = require('./backstage-layout')
, BackstageLoginComponent = require('./backstage-login')
, ArvApiDirectoryComponent = require('./component.arv-api-directory')
, ArvIndexComponent = require('./component.arv-index')
, ArvShowComponent = require('./component.arv-show');

window.jQuery = require('jquery');
require('bootstrap');

var connections = m.prop('4xphq a855m c97qk qr1hi 9tee4 su92l tb05z wx7k5'.split(' ').map(
    function(site) {
        return ArvadosConnection.make(site);
    }));

m.route(document.body, '/', {
    '/login-callback': new BackstageLoginComponent(),
    '/': new BackstageLayout({
        modules: {
            content: new ArvApiDirectoryComponent({connections: connections}),
        },
    }),
    '/list/:connection/:modelName': new BackstageLayout({
        modules: {
            content: ArvIndexComponent
        },
    }),
    '/show/:uuid': new BackstageLayout({
        modules: {
            content: ArvShowComponent
        },
    }),
});
