// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// SaveUIState avoids losing scroll position due to navigation
// events, and saves/restores other caller-specified UI state.
//
// It does not display any content itself: do not pass any children.
//
// Use of multiple SaveUIState components on the same page is not
// (yet) supported.
//
// The problem being solved:
//
// Page 1 loads some content dynamically (e.g., via infinite scroll)
// after the initial render. User scrolls down, clicks a link, and
// lands on page 2. User clicks the Back button, and lands on page
// 1. Page 1 renders its initial content while waiting for AJAX.
//
// But (without SaveUIState) the document body is small now, so the
// browser resets scroll position to the top of the page. Even if we
// end up displaying the same dynamic content, the user's place on the
// page has been lost.
//
// SaveUIState fixes this by stashing the current body height when
// navigating away from page 1. When navigating back, it restores the
// body height even before the page has loaded, so the browser does
// not reset the scroll position.
//
// SaveUIState also saves/restores arbitrary UI state (like text typed
// in a search box) in response to navigation events.
//
// See CollectionsSearch for an example.
//
// Attributes:
//
// {getter-setter} currentState: the current UI state
//
// {any} defaultState: value to initialize currentState with, if
// nothing is stashed in browser history.
//
// {boolean} forgetSavedHeight: the body height loaded from the
// browser history (if any) is outdated; we should let the browser
// determine the correct body height from the current page
// content. Set this when dynamic content has been reset.
//
// {boolean} saveBodyHeight: save/restore body height as described
// above.
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
        if (vnode.attrs.saveBodyHeight && vnode.attrs.forgetSavedHeight) {
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
