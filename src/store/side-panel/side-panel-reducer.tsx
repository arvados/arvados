// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sidePanelActions } from "./side-panel-action"

interface SidePanelState {
  collapsedState: boolean
}

const sidePanelInitialState = {
  collapsedState: false
}

export const sidePanelReducer = (state: SidePanelState = sidePanelInitialState, action)=>{
  if(action.type === sidePanelActions.TOGGLE_COLLAPSE) return {...state, collapsedState: action.payload}
  return state
}