// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";

import { RootState } from "../../store";
import { collectionService } from '../../../services/services';
import { CollectionResource } from '../../../models/collection';

export const collectionCreateActions = unionize({
    OPEN_COLLECTION_CREATOR: ofType<{ ownerUuid: string }>(),
    CLOSE_COLLECTION_CREATOR: ofType<{}>(),
    CREATE_COLLECTION: ofType<{}>(),
    CREATE_COLLECTION_SUCCESS: ofType<{}>(),
}, {
        tag: 'type',
        value: 'payload'
    });

export const createCollection = (collection: Partial<CollectionResource>) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { ownerUuid } = getState().collectionCreation.creator;
        const collectiontData = { ownerUuid, ...collection };
        dispatch(collectionCreateActions.CREATE_COLLECTION(collectiontData));
        return collectionService
            .create(collectiontData)
            .then(collection => dispatch(collectionCreateActions.CREATE_COLLECTION_SUCCESS(collection)));
    };

export type CollectionCreateAction = UnionOf<typeof collectionCreateActions>;