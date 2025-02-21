// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { selectedResourceActions } from "./selected-resource-actions";

type SelectedResourceState = {
    selectedResourceUuid: string | null
};

const initialState: SelectedResourceState = {
    selectedResourceUuid: null
}

export const selectedResourceReducer = (state: SelectedResourceState = initialState, action: any) => {
    if (action.type === selectedResourceActions.SET_SELECTED_RESOURCE) {
        return { ...state, selectedResourceUuid: action.payload };
    }
    return state;
};