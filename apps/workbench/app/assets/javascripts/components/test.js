// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.TestComponent = {
    view: function(vnode) {
        return m('div.mithril-test-component', [
            m('p', {
                onclick: m.withAttr('zzz', function(){}),
            }, [
                'mithril is working; rendered at t=',
                (new Date()).getTime(),
                'ms (click to re-render)',
            ]),
        ])
    },
}
