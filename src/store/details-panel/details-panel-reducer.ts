// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { detailsPanelActions, DetailsPanelAction } from "./details-panel-action";

export interface DetailsPanelState {
    resourceUuid: string;
    isOpened: boolean;
    tabNr: number;
}

const initialState = {
    resourceUuid: '',
    isOpened: false,
    tabNr: 0
};

export const detailsPanelReducer = (state: DetailsPanelState = initialState, action: DetailsPanelAction) =>
    detailsPanelActions.match(action, {
        default: () => state,
        LOAD_DETAILS_PANEL: resourceUuid => ({ ...state, resourceUuid }),
        OPEN_DETAILS_PANEL: tabNr => ({ ...state, isOpened: true, tabNr }),
        TOGGLE_DETAILS_PANEL: () => ({ ...state, isOpened: !state.isOpened }),
    });
