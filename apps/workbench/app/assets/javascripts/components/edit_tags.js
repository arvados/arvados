// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Plugin taken from: https://gist.github.com/thiago-negri/132bf33b5312e2da823c
// This behavior seems to be planned to be integrated on the next selectize release
Selectize.define('no_results', function(options) {
    var self = this;

    options = $
        .extend({message: 'No results found.', html: function(data) {
            return ('<div class="selectize-dropdown ' + data.classNames + ' dropdown-empty-message">' + '<div class="selectize-dropdown-content" style="padding: 3px 12px">' + data.message + '</div>' + '</div>');
        }}, options);

    self.displayEmptyResultsMessage = function() {
        this.$empty_results_container.css('top', this.$control.outerHeight());
        this.$empty_results_container.css('width', this.$control.outerWidth());
        this.$empty_results_container.show();
    };

    self.on('type', function(str) {
        if (str && !self.hasOptions) {
            self.displayEmptyResultsMessage();
        } else {
            self.$empty_results_container.hide();
        }
    });

    self.onKeyDown = (function() {
        var original = self.onKeyDown;

        return function(e) {
            original.apply(self, arguments);
            this.$empty_results_container.hide();
        }
    })();

    self.onBlur = (function() {
        var original = self.onBlur;

        return function() {
            original.apply(self, arguments);
            this.$empty_results_container.hide();
        };
    })();

    self.setup = (function() {
        var original = self.setup;
        return function() {
            original.apply(self, arguments);
            self.$empty_results_container = $(options.html(
                $.extend({classNames: self.$input.attr('class')}, options)));
            self.$empty_results_container.insertBefore(self.$dropdown);
            self.$empty_results_container.hide();
        };
    })();
});

window.SimpleInput = {
    view: function(vnode) {
        return m("input.form-control", {
            style: {
                width: '100%',
            },
            type: 'text',
            placeholder: vnode.attrs.placeholder,
            value: vnode.attrs.value,
            onchange: function() {
                console.log(this.value)
                if (this.value != '') {
                    vnode.attrs.value(this.value)
                }
            },
        }, vnode.attrs.value)
    },
    oncreate: function(vnode) {
        if (vnode.attrs.setFocus) {
            vnode.dom.focus()
        }
    }
}

window.SelectOrAutocomplete = {
    onFocus: function(vnode) {
        // Allow the user to edit an already entered value by removing it
        // and filling the input field with the same text
        var activeSelect = vnode.state.selectized[0].selectize
        value = activeSelect.getValue()
        if (value.length > 0) {
            activeSelect.clear(silent = true)
            activeSelect.setTextboxValue(value)
        }
    },
    view: function(vnode) {
        return m("input", {
            style: {
                width: '100%'
            },
            type: 'text',
            value: vnode.attrs.value
        }, vnode.attrs.value)
    },
    oncreate: function(vnode) {
        vnode.state.selectized = $(vnode.dom).selectize({
            plugins: ['no_results'],
            labelField: 'value',
            valueField: 'value',
            searchField: 'value',
            sortField: 'value',
            persist: false,
            hideSelected: true,
            openOnFocus: !vnode.attrs.create,
            createOnBlur: true,
            // selectOnTab: true,
            maxItems: 1,
            placeholder: (vnode.attrs.create ? 'Add or select ': 'Select ') + vnode.attrs.placeholder,
            create: vnode.attrs.create ? function(input) {
                return {value: input}
            } : false,
            items: [vnode.attrs.value()],
            options: vnode.attrs.options.map(function(option) {
                return {value: option}
            }),
            onChange: function(val) {
                if (val != '') {
                    vnode.attrs.value(val)
                }
            },
            onFocus: function() {
                vnode.state.onFocus(vnode)
            }
        })
        if (vnode.attrs.setFocus) {
            vnode.state.selectized[0].selectize.focus()
        }
    }
}

window.TagEditorRow = {
    view: function(vnode) {
        var nameOpts = Object.keys(vnode.attrs.vocabulary().tags)
        var valueOpts = []
        var inputComponent = SelectOrAutocomplete
        if (nameOpts.length === 0) {
            // If there's not vocabulary defined, switch to a simple input field
            inputComponent = SimpleInput
        } else {
            // Name options list
            if (vnode.attrs.name() != '' && !(vnode.attrs.name() in vnode.attrs.vocabulary().tags)) {
                nameOpts.push(vnode.attrs.name())
            }
            // Value options list
            if (vnode.attrs.name() in vnode.attrs.vocabulary().tags &&
                'values' in vnode.attrs.vocabulary().tags[vnode.attrs.name()]) {
                    valueOpts = vnode.attrs.vocabulary().tags[vnode.attrs.name()].values
            }
            if (vnode.attrs.value() != '') {
                valueOpts.push(vnode.attrs.value())
            }
        }
        return m("tr", [
            // Erase tag
            m("td", [
                vnode.attrs.editMode &&
                m('div.text-center', m('a.btn.btn-default.btn-sm', {
                    style: {
                        align: 'center'
                    },
                    onclick: function(e) { vnode.attrs.removeTag() }
                }, m('i.fa.fa-fw.fa-trash-o')))
            ]),
            // Tag name
            m("td", [
                vnode.attrs.editMode ?
                m("div", {key: 'name-'+vnode.attrs.name()},[
                    m(inputComponent, {
                        options: nameOpts,
                        value: vnode.attrs.name,
                        // Allow any tag name unless "strict" is set to true.
                        create: !vnode.attrs.vocabulary().strict,
                        placeholder: 'name',
                        // Focus on tag name field when adding a new tag that's
                        // not the first one.
                        setFocus: vnode.attrs.name() === ''
                    })
                ])
                : vnode.attrs.name
            ]),
            // Tag value
            m("td", [
                vnode.attrs.editMode ?
                m("div", {key: 'value-'+vnode.attrs.name()}, [
                    m(inputComponent, {
                        options: valueOpts,
                        value: vnode.attrs.value,
                        placeholder: 'value',
                        // Allow any value on tags not listed on the vocabulary.
                        // Allow any value on tags without values, or the ones
                        // that aren't explicitly declared to be strict.
                        create: !(vnode.attrs.name() in vnode.attrs.vocabulary().tags)
                            || !vnode.attrs.vocabulary().tags[vnode.attrs.name()].values
                            || vnode.attrs.vocabulary().tags[vnode.attrs.name()].values.length === 0
                            || !vnode.attrs.vocabulary().tags[vnode.attrs.name()].strict,
                        // Focus on tag value field when new tag name is set
                        setFocus: vnode.attrs.name() !== '' && vnode.attrs.value() === ''
                    })
                ])
                : vnode.attrs.value
            ])
        ])
    }
}

window.TagEditorTable = {
    view: function(vnode) {
        return m("table.table.table-condensed.table-justforlayout", [
            m("colgroup", [
                m("col", {width:"5%"}),
                m("col", {width:"25%"}),
                m("col", {width:"70%"}),
            ]),
            m("thead", [
                m("tr", [
                    m("th"),
                    m("th", "Key"),
                    m("th", "Value"),
                ])
            ]),
            m("tbody", [
                vnode.attrs.tags.length > 0
                ? vnode.attrs.tags.map(function(tag, idx) {
                    return m(TagEditorRow, {
                        key: idx,
                        removeTag: function() {
                            vnode.attrs.tags.splice(idx, 1)
                            vnode.attrs.dirty(true)
                        },
                        editMode: vnode.attrs.editMode,
                        name: tag.name,
                        value: tag.value,
                        vocabulary: vnode.attrs.vocabulary
                    })
                })
                : m("tr", m("td[colspan=3]", m("center","loading tags...")))
            ]),
        ])
    }
}

window.TagEditorApp = {
    appendTag: function(vnode, name, value) {
        var tag = {name: m.stream(name), value: m.stream(value)}
        vnode.state.tags.push(tag)
        // Set dirty flag when any of name/value changes to non empty string
        tag.name.map(function(v) {
            if (v !== '') {
                vnode.state.dirty(true)
            }
        })
        tag.value.map(function(v) {
            if (v !== '') {
                vnode.state.dirty(true)
            }
        })
        tag.name.map(m.redraw)
    },
    oninit: function(vnode) {
        vnode.state.sessionDB = new SessionDB()
        // Get vocabulary
        vnode.state.vocabulary = m.stream({"strict":false, "tags":{}})
        m.request('/vocabulary.json').then(vnode.state.vocabulary)
        vnode.state.editMode = vnode.attrs.targetEditable
        vnode.state.tags = []
        vnode.state.dirty = m.stream(false)
        vnode.state.dirty.map(m.redraw)
        vnode.state.objPath = '/arvados/v1/'+vnode.attrs.targetController+'/'+vnode.attrs.targetUuid
        // Get tags
        vnode.state.sessionDB.request(
            vnode.state.sessionDB.loadLocal(),
            '/arvados/v1/'+vnode.attrs.targetController,
            {
                data: {
                    filters: JSON.stringify([['uuid', '=', vnode.attrs.targetUuid]]),
                    select: JSON.stringify(['properties'])
                },
            }).then(function(obj) {
                if (obj.items.length == 1) {
                    o = obj.items[0]
                    Object.keys(o.properties).forEach(function(k) {
                        vnode.state.appendTag(vnode, k, o.properties[k])
                    })
                    // Data synced with server, so dirty state should be false
                    vnode.state.dirty(false)
                    // Add new tag row when the last one is completed
                    vnode.state.dirty.map(function() {
                        if (!vnode.state.editMode) { return }
                        lastTag = vnode.state.tags.slice(-1).pop()
                        if (lastTag === undefined || (lastTag.name() !== '' && lastTag.value() !== '')) {
                            vnode.state.appendTag(vnode, '', '')
                        }
                    })
                }
            }
        )
    },
    view: function(vnode) {
        return [
            vnode.state.editMode &&
            m("div.pull-left", [
                m("a.btn.btn-primary.btn-sm"+(vnode.state.dirty() ? '' : '.disabled'), {
                    style: {
                        margin: '10px 0px'
                    },
                    onclick: function(e) {
                        var tags = {}
                        vnode.state.tags.forEach(function(t) {
                            if (t.name() != '' && t.value() != '') {
                                tags[t.name()] = t.value()
                            }
                        })
                        vnode.state.sessionDB.request(
                            vnode.state.sessionDB.loadLocal(),
                            vnode.state.objPath, {
                                method: "PUT",
                                data: {properties: JSON.stringify(tags)}
                            }
                        ).then(function(v) {
                            vnode.state.dirty(false)
                        })
                    }
                }, vnode.state.dirty() ? ' Save changes ' : ' Saved ')
            ]),
            // Tags table
            m(TagEditorTable, {
                editMode: vnode.state.editMode,
                tags: vnode.state.tags,
                vocabulary: vnode.state.vocabulary,
                dirty: vnode.state.dirty
            })
        ]
    },
}