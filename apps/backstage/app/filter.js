module.exports = {
    AnyText: FilterAnyText,
    ObjectType: FilterObjectType,
};

var m = require('mithril');

function FilterAnyText() {
    this.view = function(ctrl) {
        return m('.input-group.input-group-sm', [
            m('input.form-control[type="text"][placeholder="Search"]',
              {value: getFilter(),
               oninput: m.withAttr('value', setFilter)}),
        ]);
        function getFilter() {
            return ctrl.currentFilter() ? ctrl.currentFilter()[2].replace(/^%(.*)%$/, '$1') : ''
        }
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
                        ctrl.currentFilter() ? ctrl.currentFilter()[2].replace(/^.*#/,'') : 'Any type',
                        ' ',
                        m('span.caret'),
                    ]),
                    m('ul.dropdown-menu[role="menu"]', [
                        m('li', [
                            m('a[href="#"][data-value="arvados#collection"]', {onclick: ctrl.currentFilter.bind(ctrl, opts.attr, 'is_a', 'arvados#collection')}, 'Collection'),
                        ]),
                        m('li', [
                            m('a[href="#"][data-value="arvados#pipelineInstance"]', {onclick: ctrl.currentFilter.bind(ctrl, opts.attr, 'is_a', 'arvados#pipelineInstance')}, 'Pipeline instance'),
                        ]),
                    ]),
                ]),
            ]),
        ];
    };
}
