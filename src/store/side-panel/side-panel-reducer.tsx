// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sidePanelActions } from "./side-panel-action"

interface SidePanelState {
  collapsedState: boolean,
  currentSideWidth: number
}

const sidePanelInitialState = {
  collapsedState: false,
  currentSideWidth: 0
}

export const sidePanelReducer = (state: SidePanelState = sidePanelInitialState, action)=>{
  if(action.type === sidePanelActions.TOGGLE_COLLAPSE) return {...state, collapsedState: action.payload}
  if(action.type === sidePanelActions.SET_CURRENT_WIDTH) return {...state, currentSideWidth: action.payload}
  return state
}