// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const selectedResourceActions = {
    SET_SELECTED_RESOURCE: 'SET_SELECTED_RESOURCE',
}

type SelectedResourceAction = {
    type: string;
    payload: string;
};

export const setSelectedResourceUuid = (resourceUuid: string): SelectedResourceAction => ({
    type: selectedResourceActions.SET_SELECTED_RESOURCE,
    payload: resourceUuid
});
