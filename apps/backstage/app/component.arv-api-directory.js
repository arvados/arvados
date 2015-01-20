module.exports = ArvApiDirectoryComponent;

var m = require('mithril');
var _ = require('lodash');
var BaseController = require('./base-ctrl');
var ArvApiStatusComponent = require('./component.arv-api-status');

function ArvApiDirectoryComponent(opts) {
    _.extend(this, {
        controller: this.controller.bind(this, opts),
    });
}
_.extend(ArvApiDirectoryComponent.prototype, {
    controller: controller,
    view: view,
});
function controller(opts) {
    _.extend(this, {connections: m.prop([])}, opts);
    this.redrawTimer = setInterval(function() {
        // If redraw is really really cheap, we can do this to make
        // "#seconds old" timers count in real time.
        m.redraw();
    }, 1000);
    this.widgets = this.connections().map(function(conn) {
        var component = new ArvApiStatusComponent(conn);
        return {
            view: component.view,
            controller: new component.controller(),
        };
    });
}
_.extend(controller.prototype, BaseController.prototype, {
    // Give BaseController a list of components to unload.
    controllers: function controllers() {
        return this.widgets.map(function(widget) {
            return widget.controller;
        });
    },
    onunload: function onunload() {
        clearTimeout(this.redrawTimer);
        BaseController.prototype.onunload.call(this);
    },
});
function view(ctrl) {
    return m('div', [
        ctrl.widgets.map(function(widget) {
            return widget.view(widget.controller);
        })
    ]);
}
