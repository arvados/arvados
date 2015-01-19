module.exports = ArvIndexComponent;

var m = require('mithril')
, BaseController = require('app/base-ctrl')
, ArvListComponent = require('app/component.arv-list')
, ArvObjectRowComponent = require('app/component.arv-object-row')
, InfiniteScroll = require('app/infinitescroll');

function ArvIndexComponent() {}
ArvIndexComponent.controller = function controller() {
    this.list =
        new ArvListComponent(null, null, ArvObjectRowComponent);
    this.listCtrl =
        new this.list.controller();
    this.scroller =
        new InfiniteScroll(this.listCtrl, this.list.view, {pxThreshold: 200});
    this.scrollerCtrl =
        new this.scroller.controller();
};
ArvIndexComponent.controller.prototype.controllers = function controllers() {
    return [this.listCtrl, this.scrollerCtrl];
};
ArvIndexComponent.view = function view(ctrl) {
    return ctrl.scroller.view(ctrl.scrollerCtrl);
};
