// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";

import { RootState } from "../../store";
import { collectionService } from '../../../services/services';
import { CollectionResource } from '../../../models/collection';

export const collectionUpdatorActions = unionize({
    OPEN_COLLECTION_UPDATOR: ofType<{ ownerUuid: string }>(),
    CLOSE_COLLECTION_UPDATOR: ofType<{}>(),
    UPDATE_COLLECTION: ofType<{}>(),
    UPDATE_COLLECTION_SUCCESS: ofType<{}>(),
}, {
        tag: 'type',
        value: 'payload'
    });

export const updateCollection = (collection: Partial<CollectionResource>) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { ownerUuid } = getState().collections.creator;
        const collectiontData = { ownerUuid, ...collection };
        dispatch(collectionUpdatorActions.UPDATE_COLLECTION(collectiontData));
        return collectionService
            // change for update
            .create(collectiontData)
            .then(collection => dispatch(collectionUpdatorActions.UPDATE_COLLECTION_SUCCESS(collection)));
    };

export type CollectionUpdatorAction = UnionOf<typeof collectionUpdatorActions>;