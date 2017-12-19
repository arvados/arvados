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
        // Value options list
        valueOpts = []
        if (vnode.attrs.name() in vnode.attrs.vocabulary().types &&
            'options' in vnode.attrs.vocabulary().types[vnode.attrs.name()]) {
                valueOpts = vnode.attrs.vocabulary().types[vnode.attrs.name()].options
        }
        valueOpts.push(vnode.attrs.value())

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
                options: valueOpts,
                value: vnode.attrs.value,
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
                vnode.attrs.tags.map(function(tag, idx) {
                    return m(TagEditorRow, {
                        key: idx,
                        removeTag: function() { vnode.attrs.tags.splice(idx, 1) },
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
        vnode.state.saveLabel = m.stream(' Save ')
        vnode.state.sessionDB = new SessionDB()
        // Get vocabulary
        vnode.state.vocabulary = m.stream({"strict":false, "types":{}})
        m.request('/vocabulary.json').then(vnode.state.vocabulary)
        vnode.state.editMode = vnode.attrs.targetEditable
        // Get tags
        vnode.state.tags = []
        vnode.state.objPath = '/arvados/v1/'+vnode.attrs.targetController+'/'+vnode.attrs.targetUuid
        vnode.state.sessionDB.request(
            vnode.state.sessionDB.loadLocal(), vnode.state.objPath, {
                data: {
                    select: JSON.stringify(['properties']) // FIXME: not working
                },
            }).then(function(obj) {
                console.log(obj)
                Object.keys(obj.properties).forEach(function(k) {
                    vnode.state.tags.push({
                        name: m.stream(k),
                        value: m.stream(obj.properties[k])
                    })
                })
                vnode.state.dirty = m.stream(null)
                vnode.state.tags.map(function(tag) {
                  tag.name.map(m.redraw)
                  tag.name.map(vnode.state.dirty)
                  tag.value.map(vnode.state.dirty)
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
            vnode.state.editMode &&
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
                            vnode.state.saveLabel('Saving...')
                            var tags = {}
                            vnode.state.tags.forEach(function(t) {
                                tags[t.name()] = t.value()
                            })
                            vnode.state.sessionDB.request(
                                vnode.state.sessionDB.loadLocal(),
                                vnode.state.objPath, {
                                    method: "PUT",
                                    data: {properties: JSON.stringify(tags)}
                                }
                            ).then(function(v) {
                                vnode.state.saveLabel(' Save ')
                                console.log('ok!')
                            })
                        }
                    }, vnode.state.saveLabel)
                ])
            ])
        ]
    },
}