// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionUpdaterActions, CollectionUpdaterAction } from './collection-updater-action';

export interface CollectionUpdaterState {
    opened: boolean;
    uuid: string;
}

const updateCollection = (state: CollectionUpdaterState, updater?: Partial<CollectionUpdaterState>) => ({
    ...state,
    ...updater
});

const initialState: CollectionUpdaterState = {
    opened: false,
    uuid: ''
};

export const collectionUpdaterReducer = (state: CollectionUpdaterState = initialState, action: CollectionUpdaterAction) => {
    return collectionUpdaterActions.match(action, {
        OPEN_COLLECTION_UPDATER: ({ uuid }) => updateCollection(state, { uuid, opened: true }),
        CLOSE_COLLECTION_UPDATER: () => updateCollection(state, { opened: false }),
        UPDATE_COLLECTION_SUCCESS: () => updateCollection(state, { opened: false, uuid: "" }),
        default: () => state
    });
};
