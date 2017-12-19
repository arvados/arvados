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
        vnode.state.selector = $(vnode.dom).selectize({
            labelField: 'value',
            valueField: 'value',
            searchField: 'value',
            sortField: 'value',
            maxItems: 1,
            create: vnode.attrs.create ? function(input) {
                return {value: input}
            } : false,
            items: [vnode.attrs.value()],
            options: vnode.attrs.options.map(function(option) {
                return {value: option}
            }),
            onChange: function(val) {
                vnode.state.dirty = true
                vnode.attrs.value(val)
                m.redraw()
            }
        }).data('selectize')
    }
}

// When in edit mode, present a tag name selector and tag value
// selector/editor depending of the tag type.
window.TagEditorRow = {
    view: function(vnode) {
        return m("tr", [
            // Erase tag
            m("td",
            vnode.attrs.editMode ?
            m("i.glyphicon.glyphicon-remove", {
                style: "cursor: pointer;",
                onclick: function(e) {
                    console.log('Erase tag clicked')
                    vnode.attrs.removeTag()
                }
            })
            : ""),
            // Tag name
            m("td",
            vnode.attrs.editMode ?
            m("div", {key: 'name-'+vnode.attrs.name()},[m(SelectOrAutocomplete, {
                options: (vnode.attrs.name() in vnode.attrs.vocabulary().types)
                    ? Object.keys(vnode.attrs.vocabulary().types)
                    : Object.keys(vnode.attrs.vocabulary().types).concat(vnode.attrs.name()),
                value: vnode.attrs.name,
                create: vnode.attrs.vocabulary().strict
            })])
            : vnode.attrs.name),
            // Tag value
            m("td",
            vnode.attrs.editMode ?
            m("div", {key: 'value-'+vnode.attrs.name()}, [m(SelectOrAutocomplete, {
                options: (vnode.attrs.name() in vnode.attrs.vocabulary().types) 
                    ? vnode.attrs.vocabulary().types[vnode.attrs.name()].options.concat(vnode.attrs.value())
                    : [vnode.attrs.value()],
                value: vnode.attrs.value,
                create: (vnode.attrs.name() in vnode.attrs.vocabulary().types)
                    ? vnode.attrs.vocabulary().types[vnode.attrs.name()].overridable || false
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
                vnode.attrs.tags.map(function(tag, idx) {
                    return m(TagEditorRow, {
                        key: idx,
                        removeTag: function() {
                            vnode.attrs.tags.splice(idx, 1)
                        },
                        editMode: vnode.attrs.editMode,
                        name: tag.name,
                        value: tag.value,
                        vocabulary: vnode.attrs.vocabulary
                    })
                })
            ]),
        ])
    }
}

window.TagEditorApp = {
    oninit: function(vnode) {
        vnode.state.sessionDB = new SessionDB()
        // Get vocabulary
        vnode.state.vocabulary = m.stream({"strict":false, "types":{}})
        m.request('/vocabulary.json').then(vnode.state.vocabulary)
        vnode.state.editMode = vnode.attrs.targetEditable
        // Get tags
        vnode.state.tags = []
        var objPath = '/arvados/v1/'+vnode.attrs.targetController+'/'+vnode.attrs.targetUuid
        vnode.state.sessionDB.request(
            vnode.state.sessionDB.loadLocal(), objPath).then(function(obj) {
                Object.keys(obj.properties).forEach(function(k) {
                    vnode.state.tags.push({
                        name: m.stream(k),
                        value: m.stream(obj.properties[k])
                    })
                })
            }
        )
    },
    view: function(vnode) {
        return [
            // Tags table
            m(TagEditorTable, {
                editMode: vnode.state.editMode,
                tags: vnode.state.tags,
                vocabulary: vnode.state.vocabulary
            }),
            vnode.state.editMode ?
            m("div", [
                m("div.pull-left", [
                    // Add tag button
                    m("a.btn.btn-primary.btn-sm", {
                        onclick: function(e) {
                            vnode.state.tags.push({
                                name: m.stream('new tag'),
                                value: m.stream('new tag value')
                            })
                        }
                    }, [
                        m("i.glyphicon.glyphicon-plus"),
                        " Add new tag "
                    ])
                ]),
                m("div.pull-right", [
                    // Save button
                    m("a.btn.btn-primary.btn-sm", {
                        onclick: function(e) {
                            console.log('Save button clicked')
                            // vnode.state.tags.save().then(function() {
                            //     vnode.state.tags.load()
                            // })
                        }
                    }, " Save ")
                ])
            ])
            : ""
        ]
    },
}