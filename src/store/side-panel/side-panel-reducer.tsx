// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sidePanelActions } from "./side-panel-action"

const sidePanelInitialState = {
  collapsedState: false
}

export const sidePanelReducer = (state = sidePanelInitialState, action)=>{
  if(action.type === sidePanelActions.TOGGLE_COLLAPSE) return {...state, collapsedState: action.payload}
  return state
}