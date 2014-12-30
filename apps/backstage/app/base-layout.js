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

var BaseComponent = require('app/base-component')
, BaseController = require('app/base-ctrl');

Layout.prototype = BaseComponent;
function Layout(layoutView, innerModules) {
    var layout = this;
    this.views = {};
    this.controller = function controller() {
        this.controllers = [];
        Object.keys(innerModules).map(function(key) {
            var module = innerModules[key];
            var component = (module.controller instanceof Function) ? module : new module();
            var ctrl = new component.controller();
            var view = component.view.bind(component.view, ctrl);
            this.controllers.push(ctrl);
            layout.views[key] = view;
        }, this);
    };
    this.controller.prototype = new BaseController();
    this.view = layoutView.bind(this, this.controller);
}
