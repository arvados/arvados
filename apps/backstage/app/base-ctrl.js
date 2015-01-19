module.exports = BaseController;

var m = require('mithril');

function BaseController() {}
BaseController.prototype.selectUuid =
    function selectUuid(uuid) {
        m.route('/show/' + uuid);
    }
BaseController.prototype.onunload =
    function onunload() {
        var todo = [];
        if (this.controllers instanceof Function) {
            todo = this.controllers();
        } else if (this.controllers instanceof Array) {
            todo = this.controllers;
        }
        todo.map(function(ctrl) {
            if (ctrl.onunload instanceof Function)
                ctrl.onunload();
        });
    }
