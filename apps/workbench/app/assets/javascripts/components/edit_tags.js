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
        $(vnode.dom).selectize({
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
    }
}

window.TagEditorRow = {
    view: function(vnode) {
        // Name options list
        var nameOpts = Object.keys(vnode.attrs.vocabulary().types)
        if (vnode.attrs.name() != '' && !(vnode.attrs.name() in vnode.attrs.vocabulary().types)) {
            nameOpts.push(vnode.attrs.name())
        }
        // Value options list
        var valueOpts = []
        if (vnode.attrs.name() in vnode.attrs.vocabulary().types &&
            'options' in vnode.attrs.vocabulary().types[vnode.attrs.name()]) {
                valueOpts = vnode.attrs.vocabulary().types[vnode.attrs.name()].options
        }
        if (vnode.attrs.value() != '') {
            valueOpts.push(vnode.attrs.value())
        }
        return m("tr", [
            // Erase tag
            m("td",
            vnode.attrs.editMode &&
                m('div.text-center', m('a.btn.btn-default.btn-sm', {
                    style: {
                        align: 'center'
                    },
                    onclick: function(e) { vnode.attrs.removeTag() }
                }, m('i.fa.fa-fw.fa-trash-o'))),
            ),
            // Tag name
            m("td",
            vnode.attrs.editMode ?
            m("div", {key: 'name-'+vnode.attrs.name()},[m(SelectOrAutocomplete, {
                options: nameOpts,
                value: vnode.attrs.name,
                create: !vnode.attrs.vocabulary().strict,
                placeholder: 'new tag',
            })])
            : vnode.attrs.name),
            // Tag value
            m("td",
            vnode.attrs.editMode ?
            m("div", {key: 'value-'+vnode.attrs.name()}, [m(SelectOrAutocomplete, {
                options: valueOpts,
                value: vnode.attrs.value,
                placeholder: 'new value',
                create: (vnode.attrs.name() in vnode.attrs.vocabulary().types)
                    ? (vnode.attrs.vocabulary().types[vnode.attrs.name()].type == 'text') || 
                        vnode.attrs.vocabulary().types[vnode.attrs.name()].overridable || false
                    : true, // If tag not in vocabulary, we should accept any value
                })
            ])
            : vnode.attrs.value)
        ])
    }
}

window.TagEditorTable = {
    view: function(vnode) {
        return m("table.table.table-condensed", {
            border: "1"
        }, [
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
                : m("tr", m("td[colspan=3]", m("center","(no tags)")))
            ]),
        ])
    }
}

window.TagEditorApp = {
    appendTag: function(vnode, name, value) {
        var tag = {name: m.stream(name), value: m.stream(value)}
        tag.name.map(vnode.state.dirty)
        tag.value.map(vnode.state.dirty)
        tag.name.map(m.redraw)
        vnode.state.tags.push(tag)
    },
    oninit: function(vnode) {
        vnode.state.sessionDB = new SessionDB()
        // Get vocabulary
        vnode.state.vocabulary = m.stream({"strict":false, "types":{}})
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
                }
            }
        )
    },
    view: function(vnode) {
        return [
            vnode.state.editMode &&
            m("div.pull-left", [
                m("a.btn.btn-primary.btn-sm"+(!(vnode.state.dirty() === false) ? '' : '.disabled'), {
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
                }, !(vnode.state.dirty() === false) ? ' Save changes ' : ' Saved ')
            ]),
            // Tags table
            m(TagEditorTable, {
                editMode: vnode.state.editMode,
                tags: vnode.state.tags,
                vocabulary: vnode.state.vocabulary,
                dirty: vnode.state.dirty
            }),
            vnode.state.editMode &&
            m("div.pull-left", [
                // Add tag button
                m("a.btn.btn-primary.btn-sm", {
                    onclick: function(e) {
                        vnode.state.appendTag(vnode, '', '')
                    }
                }, [
                    m("i.glyphicon.glyphicon-plus"),
                    " Add new tag "
                ])
            ])
        ]
    },
}