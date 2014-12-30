module.exports = FilterSet;

var m = require('mithril');

function FilterSet(viewModules) {
    var filterSet = {};
    filterSet.vm = {};
    filterSet.controller = function(callerCtrl) {
        var ctrl = this;
        ctrl.vm = filterSet.vm;
        ctrl.vm.views = viewModules.map(function(modInfo) {
            var view = (new modInfo[1](modInfo[2])).view;
            var boundGettersetter = callerCtrl.currentFilter.bind(
                callerCtrl, modInfo[0]);
            return view.bind(view, {currentFilter: boundGettersetter});
        });
    };
    filterSet.view = function(ctrl) {
        return m('form.form-inline', [
            ctrl.vm.views.map(function(view) {
                return [view(), ' '];
            }),
        ]);
    };
    return filterSet;
}
