module.exports = ArvIndexComponent;

var m = require('mithril');
var _ = require('lodash');
var BaseController = require('./base-ctrl');
var ArvListComponent = require('./component.arv-list')
var ArvObjectRowComponent = require('./component.arv-object-row')
var InfiniteScroll = require('./infinitescroll');

function ArvIndexComponent() {}
_.extend(ArvIndexComponent.prototype, {
    controller: controller,
    view: view,
});
function controller() {
    this.list =
        new ArvListComponent(null, null, ArvObjectRowComponent);
    this.listCtrl =
        new this.list.controller();
    this.scroller =
        new InfiniteScroll(this.listCtrl, this.list.view, {pxThreshold: 200});
    this.scrollerCtrl =
        new this.scroller.controller();
}
_.extend(controller.prototype, BaseController.prototype, {
    controllers: function controllers() {
        return [this.listCtrl, this.scrollerCtrl];
    },
});
function view(ctrl) {
    return ctrl.scroller.view(ctrl.scrollerCtrl);
}
