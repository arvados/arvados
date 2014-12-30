// Render an arvados object as a <div class="row">.
//
// Usage:
// x = m.prop({}); // fill in [later] using ArvConnection.find, etc.
// mod = new ArvObjectRowComponent();
// ctrl = new mod.controller({item: x});
// mod.view(ctrl)
module.exports = ArvObjectRowComponent;

var m = require('mithril')
, BaseComponent = require('app/base-component');

ArvObjectRowComponent.prototype = new BaseComponent();
function ArvObjectRowComponent() {
    this.view = function(ctrl) {
        var _item = ctrl.vm.item();
        var _owner = _item.owner_uuid ? _item._conn.find(_item.owner_uuid)() : '';
        return m('.row', [
            m('.col-sm-3', [
                m('.btn.btn-xs',
                  {onclick: ctrl.selectUuid.bind(ctrl, _item.uuid)}, [
                      m('span.glyphicon.glyphicon-link'),
                  ]),
                _item.uuid,
            ]),
            m('.col-sm-4', _item.name),
            m('.col-sm-2', [
                m('a[href="/show/'+_item.owner_uuid+'"]', {config:m.route}, [
                    _owner && (_owner.full_name || _owner.name)
                ]),
            ]),
            m('.col-sm-2', new Date(_item.created_at).toLocaleString()),
        ]);
    };
}
