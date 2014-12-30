module.exports = ArvShowComponent;

var ArvadosConnection = require('arvados/client')
, m = require('mithril');

function ArvShowComponent() {
    this.controller = function() {
        this.vm = (function() {
            var vm = {};
            vm.uuid = m.route.param('uuid');
            vm.connection = ArvadosConnection.make(vm.uuid.slice(0,5));
            vm.model = vm.connection.find(vm.uuid);
            return vm;
        })();
    };
    this.view = function(ctrl) {
        return [
            m('.row', [m('.col-sm-12', ctrl.vm.uuid)]),
            Object.keys(ctrl.vm.model() || {}).map(function(key) {
                return m('.row', [
                    m('.col-sm-2.lighten', key),
                    m('.col-sm-10', ctrl.vm.model()[key]),
                ]);
            }),
        ];
    };
}
