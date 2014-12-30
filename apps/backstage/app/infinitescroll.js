// Call the content module's getMoreItems() action whenever the bottom
// edge of the content view is [within pxThreshold of being] visible.
//
// If getMoreItems() returns a promise, bottom edge visibility will be
// tested again when that promise is resolved. This should be used
// whenever getMoreItems() adds any new items, to cover the case where
// the bottom of the content view is still visible after a new page of
// items is rendered.
//
// It is the responsibility of getMoreItems() to ignore subsequent
// calls while it's busy retrieving or preparing additional content.
module.exports = InfiniteScroll;

var m = require('mithril')
, jQuery = require('jquery');

function InfiniteScroll(contentCtrl, contentView, opts) {
    var scroller = {};
    opts = opts || {};
    scroller.controller = function() {
        this.contentCtrl = contentCtrl;
        this.getMoreItems = this.contentCtrl.getMoreItems.bind(this.contentCtrl);
        this.pxThreshold = opts.pxThreshold || 0;
        this.onunload = onunload.bind(this);
        function onunload () {
            var i=0;
            InfiniteScroll.controllers().map(function(ctrl) {
                if (ctrl === this) {
                    InfiniteScroll.elements().splice(i, 1);
                    InfiniteScroll.controllers().splice(i, 1);
                } else {
                    i++;
                }
            }.bind(this));
        };
    };
    scroller.view = function(ctrl) {
        return m('.container', {config: function(el, isInit, ctx) {
            return scroller.configEl(el, isInit, ctx, ctrl);
        }}, [
            contentView(ctrl.contentCtrl)
        ]);
    };
    scroller.configEl = function(el, isInit, ctx, ctrl) {
        if (isInit) return;
        if (InfiniteScroll.elements().indexOf(el) < 0) {
            InfiniteScroll.elements().push(el);
            InfiniteScroll.controllers().push(ctrl);
        }
    };
    return scroller;
}
InfiniteScroll.elements = m.prop([]);
InfiniteScroll.controllers = m.prop([]);

(function() {
    function scrollHandler(event) {
        InfiniteScroll.elements().map(function(el, i) {
            var ctrl = InfiniteScroll.controllers()[i];
            var pxBeforeEnd =
                el.getBoundingClientRect().bottom -
                document.documentElement.clientHeight;
            var promised;
            if (pxBeforeEnd > ctrl.pxThreshold)
                return;
            if ((promised = ctrl.getMoreItems()) && promised.then)
                promised.then(scrollHandler);
        });
    }
    jQuery(window).on('DOMContentLoaded load resize scroll', scrollHandler);
})();
