// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.SimpleInput = {
    view: function(vnode) {
        return m("input.form-control", {
            style: {
                width: '100%',
            },
            type: 'text',
            placeholder: 'Add ' + vnode.attrs.placeholder,
            value: vnode.attrs.value,
            onchange: function() {
                if (this.value != '') {
                    vnode.attrs.value(this.value)
                }
            },
        }, vnode.attrs.value)
    },
}

window.SelectOrAutocomplete = {
    view: function(vnode) {
        return m("input.form-control", {
            style: {
                width: '100%'
            },
            type: 'text',
            value: vnode.attrs.value,
            placeholder: (vnode.attrs.create ? 'Add or select ': 'Select ') + vnode.attrs.placeholder,
        }, vnode.attrs.value)
    },
    oncreate: function(vnode) {
        vnode.state.awesomplete = new Awesomplete(vnode.dom, {
            list: vnode.attrs.options,
            minChars: 0,
            maxItems: 1000000,
            autoFirst: true,
            sort: false,
        })
        vnode.state.create = vnode.attrs.create
        vnode.state.options = vnode.attrs.options
        // Option is selected from the list.
        $(vnode.dom).on('awesomplete-selectcomplete', function(event) {
            vnode.attrs.value(this.value)
        })
        $(vnode.dom).on('change', function(event) {
            if (!vnode.state.create && !(this.value in vnode.state.options)) {
                this.value = vnode.attrs.value()
            } else {
                if (vnode.attrs.value() !== this.value) {
                    vnode.attrs.value(this.value)
                }
            }
        })
        $(vnode.dom).on('focusin', function(event) {
            if (this.value === '') {
                vnode.state.awesomplete.evaluate()
                vnode.state.awesomplete.open()
            }
        })
    },
    onupdate: function(vnode) {
        vnode.state.awesomplete.list = vnode.attrs.options
        vnode.state.create = vnode.attrs.create
        vnode.state.options = vnode.attrs.options
    },
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
            // Tag key
            m("td", [
                vnode.attrs.editMode ?
                m("div", {key: 'key'}, [
                    m(inputComponent, {
                        options: nameOpts,
                        value: vnode.attrs.name,
                        // Allow any tag name unless "strict" is set to true.
                        create: !vnode.attrs.vocabulary().strict,
                        placeholder: 'key',
                    })
                ])
                : vnode.attrs.name
            ]),
            // Tag value
            m("td", [
                vnode.attrs.editMode ?
                m("div", {key: 'value'}, [
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
                        key: tag.rowKey,
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
                : m("tr", m("td[colspan=3]", m("center", "Loading tags...")))
            ]),
        ])
    }
}

var uniqueID = 1

window.TagEditorApp = {
    appendTag: function(vnode, name, value) {
        var tag = {name: m.stream(name), value: m.stream(value), rowKey: uniqueID++}
        vnode.state.tags.push(tag)
        // Set dirty flag when any of name/value changes to non empty string
        tag.name.map(function() { vnode.state.dirty(true) })
        tag.value.map(function() { vnode.state.dirty(true) })
        tag.name.map(m.redraw)
    },
    oninit: function(vnode) {
        vnode.state.sessionDB = new SessionDB()
        // Get vocabulary
        vnode.state.vocabulary = m.stream({"strict":false, "tags":{}})
        var vocabularyTimestamp = parseInt(Date.now() / 300000) // Bust cache every 5 minutes
        m.request('/vocabulary.json?v=' + vocabularyTimestamp).then(vnode.state.vocabulary)
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
                    if (vnode.state.editMode) {
                        vnode.state.appendTag(vnode, '', '')
                    }
                    // Data synced with server, so dirty state should be false
                    vnode.state.dirty(false)
                    // Add new tag row when the last one is completed
                    vnode.state.dirty.map(function() {
                        if (!vnode.state.editMode) { return }
                        lastTag = vnode.state.tags.slice(-1).pop()
                        if (lastTag === undefined || (lastTag.name() !== '' || lastTag.value() !== '')) {
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
                            // Only ignore tags with empty key
                            if (t.name() != '') {
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
