// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

$(document).on('ready arv:pane:loaded', function() {
    $('[data-mount-mithril]').each(function() {
        var data = $(this).data()
        m.mount(this, {view: function () {return m(window[data.mountMithril], data)}})
    })
})
