// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionCreateActions, CollectionCreateAction } from './collection-creator-action';

export type CollectionCreatorState = CollectionCreator;

interface CollectionCreator {
    opened: boolean;
    ownerUuid: string;
}

const updateCreator = (state: CollectionCreatorState, creator?: Partial<CollectionCreator>) => ({
    ...state,
    ...creator
});

const initialState: CollectionCreatorState = {
    opened: false,
    ownerUuid: ''
};

export const collectionCreatorReducer = (state: CollectionCreatorState = initialState, action: CollectionCreateAction) => {
    return collectionCreateActions.match(action, {
        OPEN_COLLECTION_CREATOR: ({ ownerUuid }) => updateCreator(state, { ownerUuid, opened: true }),
        CLOSE_COLLECTION_CREATOR: () => updateCreator(state, { opened: false }),
        CREATE_COLLECTION: () => updateCreator(state),
        CREATE_COLLECTION_SUCCESS: () => updateCreator(state, { opened: false, ownerUuid: "" }),
        default: () => state
    });
};
