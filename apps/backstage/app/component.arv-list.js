module.exports = ArvListComponent;

var m = require('mithril')
, ArvadosClient = require('arvados/client')
, BaseController = require('app/base-ctrl')
, Filter = require('app/filter')
, FilterSet = require('app/filterset')
, util = require('app/util');

function ArvListComponent(connection, arvModelName, contentModule) {
    this.controller = Controller;
    this.view = View;

    Controller.prototype = new BaseController();
    function Controller() {
        this.vm = new ViewModel();
        this.vm.init();
        this.vm.filterSetCtrl = new this.vm.filterSet.controller(this);
    }
    Controller.prototype.getMoreItems =
        function getMoreItems() {
            return this.vm.getMoreItems.apply(this.vm, arguments);
        }
    Controller.prototype.currentFilter =
        function currentFilter(key, attr, operator, operand) {
            if (arguments.length > 1) {
                this.vm.filters[key] = [attr, operator, operand];
                util.debounce(500, this.vm.resetContent).
                    then(this.vm.resetContent);
            }
            return this.vm.filters[key];
        }

    function ViewModel() {
        var vm = this;
        vm.init = function() {
            vm.arvModelName = arvModelName || m.route.param('modelName');
            vm.connection = connection || ArvadosClient.make(m.route.param('connection'));
            vm.filters = {};
            vm.filterSet = new FilterSet(
                [['any', Filter.AnyText],
                 ['type', Filter.ObjectType, {attr:'uuid'}]]);
            vm.inflight = null;
            vm.listLimit = 30;
            vm.listOrders = ['created_at desc'];
            vm.resetContent();
        };
        vm.resetContent = function() {
            if (vm.inflight) {
                // Forget about current/stale request. TODO: abort the xhr.
                vm.inflight.reject();
                vm.inflight = null;
            }
            vm.eof = m.prop(false);
            vm.items = m.prop([]);
            vm.itemViews = m.prop([]);
            vm.beforeRender = function() {
                // On first render, trigger a scroll event to make the
                // first page of content appear. The scroll handler can
                // ignore this if (for example) the content view is
                // invisible now.
                vm.beforeRender = function() {};
                window.setTimeout(function() {
                    window.dispatchEvent(new Event('scroll'));
                }, 1);
            };
        };
        vm.apiFilters = function() {
            var filters = [];
            Object.keys(vm.filters).map(function(key) {
                if (vm.filters[key])
                    filters.push(vm.filters[key]);
            });
            return filters;
        };
        vm.makeItemViews = function() {
            vm.itemViews(vm.items().map(function(item) {
                return contentModule.view.bind(
                    contentModule.view,
                    new contentModule.controller({item:item}));
            }));
        };
        vm.getMoreItems = function() {
            var inflight;
            if (vm.inflight || vm.eof())
                return false;
            inflight = m.deferred();
            vm.connection.api(vm.arvModelName, 'list', {
                filters: vm.apiFilters(),
                limit: vm.listLimit,
                offset: vm.items().length,
                order: vm.listOrders
            }).then(function(newItems) {
                if (inflight !== vm.inflight) {
                    // This request has already been superseded by a
                    // new one. Ignore.
                    return;
                }
                vm.eof(newItems.length === 0);
                vm.items(vm.items().concat(newItems));
            }, vm.eof).then(vm.makeItemViews).then(function() {
                // Give the new items a chance to render before
                // resolving the promise. This makes it possible for
                // the resolve callback to measure the DOM after the
                // new elements have been added (notably, in order to
                // keep fetching pages until the scroll threshold is
                // satisfied).
                window.setTimeout(inflight.resolve, 50);
                vm.inflight = null;
            });
            return (vm.inflight = inflight).promise;
        };
        return vm;
    }

    function View(ctrl) {
        ctrl.vm.beforeRender();
        return [
            ctrl.vm.filterSet ? ctrl.vm.filterSet.view(ctrl.vm.filterSetCtrl) : '',
            ctrl.vm.itemViews().map(function(v) {
                return v();
            }),
            ctrl.vm.eof() ? '' : m('.row', {style: 'background: #ffffdd'}, [
                m('.col-sm-12', {style: 'text-align: center'}, ['...loading...'])
            ]),
        ];
    }
}
