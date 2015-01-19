// Layout class. Instances are suitable for passing to m.route().
//
// Usage:
// new Layout(viewFunction, {main: FooModuleClass, nav: NavComponent})
//
// viewFunction is expected to include this.views.main() and
// this.views.nav() somewhere in its return value.
//
// Content can be given as a class (in which case instances are made
// with new FooModuleClass()) or as a component instance (i.e., an
// object with a function named 'controller').
//
// The layout is responsible for creating and unloading controllers.

module.exports = Layout;

var BaseController = require('app/base-ctrl');
var _ = require('lodash');

function Layout(innerModules) {
    return _.extend(this, {
        controller: Layout.controller.bind(this, innerModules),
    });
}
Layout.controller = function controller(innerModules) {
    this.views = {};
    this.controllers = [];
    Object.keys(innerModules).map(function(key) {
        var module = innerModules[key];
        var component = (module.controller instanceof Function) ? module : new module();
        var ctrl = new component.controller();
        var view = component.view.bind(component.view, ctrl);
        this.controllers.push(ctrl);
        this.views[key] = view;
    }, this);
};
Layout.controller.prototype = new BaseController();
