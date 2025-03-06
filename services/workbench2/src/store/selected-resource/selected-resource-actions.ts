// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const selectedResourceActions = {
    SET_SELECTED_RESOURCE: 'SET_SELECTED_RESOURCE',
    SET_IS_IN_DATA_EXPLORER: 'IS_SELECTED_RESOURCE_IN_DATA_EXPLORER',
}

export const setSelectedResourceUuid = (resourceUuid: string | null) => ({
    type: selectedResourceActions.SET_SELECTED_RESOURCE,
    payload: resourceUuid
});

export const setIsSelectedResourceInDataExplorer = (isIn: boolean) => ({
    type: selectedResourceActions.SET_IS_IN_DATA_EXPLORER,
    payload: isIn
});
