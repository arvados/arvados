// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.LocalizedDateTime = {
    view: function(vnode) {
        return m('span', new Date(Date.parse(vnode.attrs.parse)).toLocaleString())
    },
}
