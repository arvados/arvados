module.exports = ArvApiDirectoryComponent;

var m = require('mithril')
, BaseComponent = require('app/base-component')
, BaseController = require('app/base-ctrl')
, ArvApiStatusComponent = require('app/component.arv-api-status');

ArvApiDirectoryComponent.prototype = new BaseComponent();
function ArvApiDirectoryComponent(connections) {
    this.controller = Controller;
    Controller.prototype = new BaseController();
    function Controller(vm) {
        this.vm = vm || {};
        this.vm.widgets = connections().map(function(conn) {
            var component = new ArvApiStatusComponent(conn);
            return {
                view: component.view,
                controller: new component.controller(),
            };
        });
        // Give BaseComponent a list of components to unload.
        this.controllers = function() {
            return this.vm.widgets.map(function(widget) {
                return widget.controller;
            });
        }.bind(this);

        this.redrawTimer = setInterval(function() {
            // If redraw is really really cheap, we can do this to make
            // "#seconds old" timers count in real time.
            m.redraw();
        }, 1000);
        this.onunload = function() {
            clearTimeout(this.redrawTimer);
        };
    };
    this.view = View;
    function View(ctrl) {
        return m('div', [
            ctrl.vm.widgets.map(function(widget) {
                return widget.view(widget.controller);
            })
        ]);
    };
}
