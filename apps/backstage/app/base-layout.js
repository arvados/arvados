// Layout class. Instances of subclasses are suitable for passing to
// m.route().
//
// Usage:
// new LayoutSubclass({modules: {main: FooModuleClass, nav: NavComponent}})
//
// Content can be given as:
// * a class -- instances are made with `new class()`
// * an array of [cls, arg] -- instances are made with `new cls.controller(arg)`
// * a component instance (i.e., an object with a function named 'controller')
//
// The layout is responsible for creating and unloading controllers.

module.exports = Layout;

var BaseController = require('./base-ctrl');
var _ = require('lodash');

function Layout(opts) {
    return _.extend(this, {
        controller: this.controller.bind(this, opts),
    });
}
_.extend(Layout.prototype, {
    controller: controller,
});
controller.prototype = new BaseController();
function controller(opts) {
    _.extend(this, {modules: {}}, opts);
    this.views = {};
    this.controllers = [];
    Object.keys(this.modules).map(function(key) {
        var module = this.modules[key];
        var component;
        var ctrl;
        if (module instanceof Array) {
            component = module[0];
            ctrl = new component.controller(module[1]);
        } else if (module.controller instanceof Function) {
            component = module;
            ctrl = new component.controller();
        } else {
            component = new module();
            ctrl = new component.controller();
        }
        var view = component.view.bind(component.view, ctrl);
        this.controllers.push(ctrl);
        this.views[key] = view;
    }, this);
}
