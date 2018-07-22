// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { detailsPanelActions, DetailsPanelAction } from "./details-panel-action";
import { Resource } from "../../models/resource";

export interface DetailsPanelState {
    item: Resource | null;
    isOpened: boolean;
}

const initialState = {
    item: null,
    isOpened: false
};

export const detailsPanelReducer = (state: DetailsPanelState = initialState, action: DetailsPanelAction) =>
    detailsPanelActions.match(action, {
        default: () => state,
        LOAD_DETAILS: () => state,
        LOAD_DETAILS_SUCCESS: ({ item }) => ({ ...state, item }),
        TOGGLE_DETAILS_PANEL: () => ({ ...state, isOpened: !state.isOpened })
    });
