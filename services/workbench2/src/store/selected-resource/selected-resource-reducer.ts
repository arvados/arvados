// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { selectedResourceActions } from "./selected-resource-actions";

type SelectedResourceState = {
    selectedResourceUuid: string | null,
    isSelectedResourceInDataExplorer: boolean
};

const initialState: SelectedResourceState = {
    selectedResourceUuid: null,
    isSelectedResourceInDataExplorer: false
}

export const selectedResourceReducer = (state: SelectedResourceState = initialState, action: any) => {
    if (action.type === selectedResourceActions.SET_SELECTED_RESOURCE) {
        return { ...state, selectedResourceUuid: action.payload };
    }
    if (action.type === selectedResourceActions.SET_IS_IN_DATA_EXPLORER) {
        return { ...state, isSelectedResourceInDataExplorer: action.payload };
    }
    return state;
};