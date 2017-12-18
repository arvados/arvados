// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Fallback vocabulary that accepts any tag type. Will be used if
// no custom vocabulary provided.
var vocabulary = {
    "strict": false,
    "types": {}
}

window.Vocabulary = function(url) {
    var v = this
    Object.assign(v, {
        url: url,
        data: {},
        load: function() {
            // Load vocabulary from rails' public directory
            m.request(v.url.origin + '/vocabulary.json').then(function(resp) {
                console.log('Vocabulary loaded')
                v.data = resp
            }).catch(function(err) {
                // Not found, use a default vocabulary
                console.log('Using default vocabulary')
                v.data = vocabulary
            })
        },
        getDef: function(tagName) {
            if (tagName in v.data.types) {
                return v.data.types[tagName]
            } else {
                return {"type": "text"} // Default 
            }
        },
        getTypes: function() {
            return Object.keys(v.data.types)
        }
    })
}

window.Tags = function(db, uuid, objType) {
    var t = this
    Object.assign(t, {
        db: db,
        uuid: uuid,
        objType: objType,
        objPath: '/arvados/v1/' + objType + '/' + uuid,
        tagIdx: 0, // Will use this as the tag access key
        data: {},
        clear: function() {
            t.data = {}
        },
        load: function() {
            // Get the tag list from the API server
            return db.request(
                db.loadLocal(), 
                t.objPath).then(function(obj){
                    t.clear()
                    Object.keys(obj.properties).map(function(k) {
                        t.addTag(k, obj.properties[k])
                    })
                }
            )
        },
        save: function() {
            return db.request(
                db.loadLocal(),
                t.objPath, {
                    method: "PUT",
                    data: {properties: JSON.stringify(t.getAll())}
                }
            )
        },
        getAll: function() {
            // return hash to be POSTed to API server
            var tags = {}
            Object.keys(t.data).map(function(k) {
                a_tag = t.data[k]
                tags[a_tag.name] = a_tag.value
            })
            return tags
        },
        addTag: function(name, value) {
            name = name || ""
            value = value || ""
            t.data[t.tagIdx] = {
                "name": name,
                "value": value
            },
            t.tagIdx++
        },
        removeTag: function(tagIdx) {
            if (tagIdx in t.data) {
                delete t.data[tagIdx]
            }
        },
        getName: function(tagIdx) {
            if (tagIdx in t.data) {
                return t.data[tagIdx].name
            }
        },
        get: function(tagIdx) {
            if (tagIdx in t.data) {
                return t.data[tagIdx]
            }
        }
    })
}
