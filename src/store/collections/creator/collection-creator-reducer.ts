// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionCreateActions, CollectionCreateAction } from './collection-creator-action';

export type CollectionCreatorState = {
    creator: CollectionCreator
};

interface CollectionCreator {
    opened: boolean;
    pending: boolean;
    ownerUuid: string;
}

const updateCreator = (state: CollectionCreatorState, creator: Partial<CollectionCreator>) => ({
    ...state,
    creator: {
        ...state.creator,
        ...creator
    }
});

const initialState: CollectionCreatorState = {
    creator: {
        opened: false,
        pending: false,
        ownerUuid: ""
    }
};

export const collectionCreationReducer = (state: CollectionCreatorState = initialState, action: CollectionCreateAction) => {
    return collectionCreateActions.match(action, {
        OPEN_COLLECTION_CREATOR: ({ ownerUuid }) => updateCreator(state, { ownerUuid, opened: true, pending: false }),
        CLOSE_COLLECTION_CREATOR: () => updateCreator(state, { opened: false }),
        CREATE_COLLECTION: () => updateCreator(state, { opened: true }),
        CREATE_COLLECTION_SUCCESS: () => updateCreator(state, { opened: false, ownerUuid: "" }),
        default: () => state
    });
};
