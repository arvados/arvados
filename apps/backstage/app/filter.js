module.exports = {
    AnyText: FilterAnyText,
    ObjectType: FilterObjectType,
};

var m = require('mithril');

function FilterAnyText() {
    this.view = function(ctrl) {
        return m('.input-group.input-group-sm', [
            m('input.form-control[type="text"][placeholder="Search"]',
              {oninput: m.withAttr('value', setFilter)}),
        ]);
        function setFilter(value) {
            ctrl.currentFilter('any', 'ilike', '%'+value+'%')
        }
    };
}

function FilterObjectType(opts) {
    this.view = function(ctrl) {
        return [
            m('.input-group.input-group-sm', [
                m('.input-group-btn', [
                    m('button.btn.btn-default.dropdown-toggle[type="button"][data-toggle="dropdown"]', [
                        ctrl.currentFilter() ? ctrl.currentFilter()[2].replace(/^.*#/,'') : 'Type',
                        ' ',
                        m('span.caret'),
                    ]),
                    m('ul.dropdown-menu[role="menu"]', [
                        m('li', [
                            m('a[href="#"]', {onclick: ctrl.currentFilter.bind(ctrl, opts.attr, 'is_a', 'arvados#collection')}, 'Collection'),
                        ]),
                        m('li', [
                            m('a[href="#"]', {onclick: ctrl.currentFilter.bind(ctrl, opts.attr, 'is_a', 'arvados#pipelineInstance')}, 'Pipeline instance'),
                        ]),
                    ]),
                ]),
            ]),
        ];
    };
}
