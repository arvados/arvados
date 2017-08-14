// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.SaveUIState = {
    saveState: function() {
        var state = history.state || {}
        state.bodyHeight = window.getComputedStyle(document.body)['height']
        state.currentState = this.currentState()
        history.replaceState(state, '')
    },
    oninit: function(vnode) {
        vnode.state.currentState = vnode.attrs.currentState
        var hstate = history.state || {}

        if (vnode.attrs.saveBodyHeight && hstate.bodyHeight) {
            document.body.style['min-height'] = hstate.bodyHeight
            delete hstate.bodyHeight
        }

        if (hstate.currentState) {
            vnode.attrs.currentState(hstate.currentState)
            delete hstate.currentState
        } else {
            vnode.attrs.currentState(vnode.attrs.defaultState)
        }

        history.replaceState(hstate, '')
    },
    oncreate: function(vnode) {
        vnode.state.saveState = vnode.state.saveState.bind(vnode.state)
        window.addEventListener('beforeunload', vnode.state.saveState)
        vnode.state.onupdate(vnode)
    },
    onupdate: function(vnode) {
        if (vnode.attrs.saveBodyHeight && vnode.attrs.forgetSavedState) {
            document.body.style['min-height'] = null
        }
    },
    onremove: function(vnode) {
        window.removeEventListener('beforeunload', vnode.state.saveState)
    },
    view: function(vnode) {
        return null
    },
}
