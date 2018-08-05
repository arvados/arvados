// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionUpdatorActions, CollectionUpdaterAction } from './collection-updater-action';

export type CollectionUpdatorState = CollectionUpdator;

interface CollectionUpdator {
    opened: boolean;
    uuid: string;
}

const updateCollection = (state: CollectionUpdatorState, updator?: Partial<CollectionUpdator>) => ({
    ...state,
    ...updator
});

const initialState: CollectionUpdatorState = {
    opened: false,
    uuid: ''
};

export const collectionCreationReducer = (state: CollectionUpdatorState = initialState, action: CollectionUpdaterAction) => {
    return collectionUpdatorActions.match(action, {
        OPEN_COLLECTION_UPDATER: ({ uuid }) => updateCollection(state, { uuid, opened: true }),
        CLOSE_COLLECTION_UPDATER: () => updateCollection(state, { opened: false }),
        UPDATE_COLLECTION_SUCCESS: () => updateCollection(state, { opened: false, uuid: "" }),
        default: () => state
    });
};
