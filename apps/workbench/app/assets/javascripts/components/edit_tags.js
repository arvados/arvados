// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.IntegerField = {
    view: function(vnode) {
        var tags = vnode.attrs.tags
        var voc = vnode.attrs.voc
        var tagName = tags.getName(vnode.attrs.tagIdx)
        var tagDef = voc.getDef(tagName)
        var min = tagDef.min || false
        var max = tagDef.max || false
        return m("input", {
            type: 'number',
            style: {
                width: '100%',
            },
            oninput: m.withAttr("value", function(val) {
                // Validations
                if (isNaN(parseInt(val))) { return }
                if (min && val < min) { return }
                if (max && val > max) { return }
                // Value accepted
                tags.data[vnode.attrs.tagIdx]["value"] = parseInt(val)
            }),
            value: tags.data[vnode.attrs.tagIdx]["value"]
        }, tags.data[vnode.attrs.tagIdx]["value"])
    }
}

window.TextField = {
    view: function(vnode) {
        var tags = vnode.attrs.tags
        var voc = vnode.attrs.voc
        var tagName = tags.getName(vnode.attrs.tagIdx)
        var tagDef = voc.getDef(tagName)
        var max_length = tagDef.max_length || false
        return m("input", {
            type: 'text',
            style: {
                width: '100%',
            },
            oninput: m.withAttr("value", function(val) {
                // Validation
                if (max_length && val.length > max_length) { return }
                // Value accepted
                tags.data[vnode.attrs.tagIdx]["value"] = val
            }),
            value: tags.data[vnode.attrs.tagIdx]["value"]
        }, tags.data[vnode.attrs.tagIdx]["value"])
    }
}

window.SelectNameField = {
    view: function(vnode) {
        return m("input[type=text]", {
            style: {
                width: '100%'
            },
        })
    },
    oncreate: function(vnode) {
        var tags = vnode.attrs.tags
        var voc = vnode.attrs.voc
        var opts = voc.getTypes().map(function(x) {
            return {
                value: x,
                label: x
            }
        })
        // Tag name not included on vocabulary, add it to the options
        var tagName = tags.getName(vnode.attrs.tagIdx)
        if (!voc.getTypes().includes(tagName)) {
            opts = opts.concat([{value: tagName, label: tagName}])
        }
        $(vnode.dom).selectize({
            options: opts,
            persist: false,
            maxItems: 1,
            labelField: 'label',
            valueField: 'value',
            items: [tags.data[vnode.attrs.tagIdx]["name"]],
            create: function(input) {
                return {
                    value: input,
                    label: input
                }
            },
            onChange: function(val) {
                tags.data[vnode.attrs.tagIdx]["name"] = val
                m.redraw()
            }
        })
    }
}

window.SelectField = {
    view: function(vnode) {
        var tags = vnode.attrs.tags
        var voc = vnode.attrs.voc
        var tagName = tags.getName(vnode.attrs.tagIdx)
        var overridable = voc.getDef(tagName).overridable || false
        var opts = voc.getDef(tagName).options
        // If current value isn't listed and it's an overridable type, add
        // it to the available options
        if (!opts.includes(tags.data[vnode.attrs.tagIdx]["value"]) &&
            overridable) {
            opts = opts.concat([tags.data[vnode.attrs.tagIdx]["value"]])
        }
        // Wrap the select inside a div element so it can be replaced
        return m("div", {
            style: {
                width: '100%'
            },
        }, [
            m("select", {
                style: {
                    width: '100%'
                },
                oncreate: function(v) {
                    $(v.dom).selectize({
                        create: overridable,
                        onChange: function(val) {
                            tags.data[vnode.attrs.tagIdx]["value"] = val
                            m.redraw() // 3rd party event handlers need to do this
                        }
                    })
                },
            }, opts.map(function(k) {
                    return m("option", {
                        value: k,
                        selected: tags.data[vnode.attrs.tagIdx]["value"] === k,
                    }, k)
                })
            )
        ])
    }
}

// Maps tag types against editor components
var typeMap = {
    "select": SelectField,
    "text": TextField,
    "integer": IntegerField
}

// When in edit mode, present a tag name selector and tag value
// selector/editor depending of the tag type.
window.TagEditor = {
    view: function(vnode) {
        var tags = vnode.attrs.tags
        var voc = vnode.attrs.voc
        var tagIdx = vnode.attrs.tagIdx
        if (tagIdx in tags.data) {
            var tagName = tags.getName(vnode.attrs.tagIdx)
            var tagType = voc.getDef(tagName).type
            return m("tr.collection-tag-"+tagName, [
                m("td",
                    vnode.attrs.editMode() ?
                    m("i.glyphicon.glyphicon-remove.collection-tag-remove", {
                        style: "cursor: pointer;",
                        onclick: function(e) {
                            // Erase tag
                            tags.removeTag(tagIdx)
                        }
                    })
                : ""),
            m("td.collection-tag-field.collection-tag-field-key",
                // Tag name
                vnode.attrs.editMode() ? 
                m(SelectNameField, {
                    tagIdx: tagIdx,
                    tags: tags,
                    voc: voc
                })
                : tags.data[tagIdx]["name"]),
            m("td.collection-tag-field.collection-tag-field-value",
                // Tag value
                vnode.attrs.editMode() ? 
                m(typeMap[tagType], {
                    tagIdx: tagIdx,
                    tags: tags,
                    voc: voc
                })
                : tags.data[tagIdx]["value"])
            ])
        }
    }
}

window.TagTable = {
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
            m("tbody.collection-tag-rows", [
                Object.keys(vnode.attrs.tags.data).map(function(k) {
                    return m(TagEditor, {
                        tagIdx: k,
                        key: k,
                        editMode: vnode.attrs.editMode,
                        tags: vnode.attrs.tags,
                        voc: vnode.attrs.voc
                    })
                })
            ]),
            ]
        )
    }
}

window.TagEditorApp = {
    oninit: function(vnode) {
        vnode.state.sessionDB = new SessionDB()
        vnode.state.url = new URL(document.URL)
        var pathname = vnode.state.url.pathname.split("/")
        vnode.state.uuid = pathname.pop()
        vnode.state.objType = pathname.pop()
        vnode.state.tags = new Tags(vnode.state.sessionDB, vnode.state.uuid, vnode.state.objType)
        vnode.state.tags.load()
        vnode.state.vocabulary = new Vocabulary(vnode.state.url)
        vnode.state.vocabulary.load()
        vnode.state.editMode = m.stream(false)
        vnode.state.tagTable = TagTable
    },
    view: function(vnode) {
        return [
            m("p", [
                // Edit button
                m("a.btn.btn-primary"+(vnode.state.editMode() ? '.disabled':''), {
                    onclick: function(e) {
                        vnode.state.editMode(true)
                    }
                }, " Edit "),
            ]),
            // Tags table
            m(vnode.state.tagTable, {
                editMode: vnode.state.editMode,
                tags: vnode.state.tags,
                voc: vnode.state.vocabulary
            }),
            vnode.state.editMode() ? 
            m("div", [
                m("div.pull-left", [
                    // Add tag button
                    m("a.btn.btn-primary.btn-sm", {
                        onclick: function(e) {
                            vnode.state.tags.addTag(vnode.state.vocabulary.getTypes()[0])
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
                            vnode.state.editMode(false)
                            vnode.state.tags.save().then(function() {
                                vnode.state.tags.load()
                            })
                        }
                    }, " Save "),
                    // Cancel button
                    m("a.btn.btn-primary.btn-sm", {
                        onclick: function(e) {
                            vnode.state.editMode(false)
                            e.redraw = false
                            vnode.state.tags.load().then(m.redraw())
                        }
                    }, " Cancel ")                    
                ])
            ])
            : ""
        ]
    },
}