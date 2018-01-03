// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.SelectOrAutocomplete = {
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
        this.selectized = $(vnode.dom).selectize({
            labelField: 'value',
            valueField: 'value',
            searchField: 'value',
            sortField: 'value',
            maxItems: 1,
            placeholder: vnode.attrs.placeholder,
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
            }
        })
        if (vnode.attrs.setFocus) {
            this.selectized[0].selectize.focus()
        }
    }
}

window.TagEditorRow = {
    view: function(vnode) {
        // Name options list
        var nameOpts = Object.keys(vnode.attrs.vocabulary().tags)
        if (vnode.attrs.name() != '' && !(vnode.attrs.name() in vnode.attrs.vocabulary().tags)) {
            nameOpts.push(vnode.attrs.name())
        }
        // Value options list
        var valueOpts = []
        if (vnode.attrs.name() in vnode.attrs.vocabulary().tags &&
            'values' in vnode.attrs.vocabulary().tags[vnode.attrs.name()]) {
                valueOpts = vnode.attrs.vocabulary().tags[vnode.attrs.name()].values
        }
        if (vnode.attrs.value() != '') {
            valueOpts.push(vnode.attrs.value())
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
                    m(SelectOrAutocomplete, {
                        options: nameOpts,
                        value: vnode.attrs.name,
                        // Allow any tag name unless "strict" is set to true.
                        create: !vnode.attrs.vocabulary().strict,
                        placeholder: 'new tag',
                        // Focus on tag name field when adding a new tag that's
                        // not the first one.
                        setFocus: !vnode.attrs.firstRow && vnode.attrs.name() === ''
                    })
                ])
                : vnode.attrs.name
            ]),
            // Tag value
            m("td", [
                vnode.attrs.editMode ?
                m("div", {key: 'value-'+vnode.attrs.name()}, [
                    m(SelectOrAutocomplete, {
                        options: valueOpts,
                        value: vnode.attrs.value,
                        placeholder: 'new value',
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
        return m("table.table.table-condensed", [
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
                        firstRow: vnode.attrs.tags.length === 1,
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