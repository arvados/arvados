// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionUpdatorActions, CollectionUpdatorAction } from './collection-updator-action';

export type CollectionUpdatorState = CollectionUpdator;

interface CollectionUpdator {
    opened: boolean;
    ownerUuid: string;
}

const updateCollection = (state: CollectionUpdatorState, updator?: Partial<CollectionUpdator>) => ({
    ...state,
    ...updator
});

const initialState: CollectionUpdatorState = {
    opened: false,
    ownerUuid: ''
};

export const collectionCreationReducer = (state: CollectionUpdatorState = initialState, action: CollectionUpdatorAction) => {
    return collectionUpdatorActions.match(action, {
        OPEN_COLLECTION_UPDATOR: ({ ownerUuid }) => updateCollection(state, { ownerUuid, opened: true }),
        CLOSE_COLLECTION_UPDATOR: () => updateCollection(state, { opened: false }),
        UPDATE_COLLECTION: () => updateCollection(state),
        UPDATE_COLLECTION_SUCCESS: () => updateCollection(state, { opened: false, ownerUuid: "" }),
        default: () => state
    });
};
