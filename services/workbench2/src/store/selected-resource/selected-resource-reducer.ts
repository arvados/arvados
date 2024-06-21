// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { selectedResourceActions } from "./selected-resource-actions";

type SelectedResourceState = string | null;

export const selectedResourceReducer = (state: SelectedResourceState = null, action: any) => {
    if (action.type === selectedResourceActions.SET_SELECTED_RESOURCE) {
        return action.payload;
    }
    return state;
};