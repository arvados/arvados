module.exports = ArvIndexComponent;

var m = require('mithril')
, BaseController = require('app/base-ctrl')
, ArvListComponent = require('app/component.arv-list')
, ArvObjectRowComponent = require('app/component.arv-object-row')
, InfiniteScroll = require('app/infinitescroll');

function ArvIndexComponent() {
    this.controller = Controller;
    this.view = function view(ctrl) {
        return ctrl.vm.scroller.view(ctrl.vm.scrollerCtrl);
    };
    function ViewModel() {
        this.list =
            new ArvListComponent(null, null, new ArvObjectRowComponent());
        this.listCtrl =
            new this.list.controller();
        this.scroller =
            new InfiniteScroll(this.listCtrl, this.list.view,
                               {pxThreshold: 200});
        this.scrollerCtrl =
            new this.scroller.controller();
    }
    function Controller() {
        this.vm = new ViewModel();
    }
    Controller.prototype = new BaseController();
    Controller.prototype.controllers =
        function controllers() {
            return [this.vm.listCtrl, this.vm.scrollerCtrl];
        }
}
