// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";

import { RootState } from "../../store";
import { CollectionResource } from '../../../models/collection';
import { ServiceRepository } from "../../../services/services";

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
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { ownerUuid } = getState().collections.creator;
        const collectiontData = { ownerUuid, ...collection };
        dispatch(collectionCreateActions.CREATE_COLLECTION(collectiontData));
        return services.collectionService
            .create(collectiontData)
            .then(collection => dispatch(collectionCreateActions.CREATE_COLLECTION_SUCCESS(collection)));
    };

export type CollectionCreateAction = UnionOf<typeof collectionCreateActions>;
