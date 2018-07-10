// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "../../common/api/common-resource-service";
import actions, { DetailsPanelAction } from "./details-panel-action";

export interface DetailsPanelState {
    item: Resource | null;
    isOpened: boolean;
}

const initialState = {
    item: null,
    isOpened: false
};

const reducer = (state: DetailsPanelState = initialState, action: DetailsPanelAction) =>
    actions.match(action, {
        default: () => state,
        LOAD_DETAILS: () => state,
        LOAD_DETAILS_SUCCESS: ({ item }) => ({ ...state, item }),
        TOGGLE_DETAILS_PANEL: () => ({ ...state, isOpened: !state.isOpened })
    });

export default reducer;
