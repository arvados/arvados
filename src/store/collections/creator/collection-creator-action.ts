// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";

import { RootState } from "../../store";
import { CollectionResource } from '~/models/collection';
import { ServiceRepository } from "~/services/services";
import { uploadCollectionFiles } from '../uploader/collection-uploader-actions';
import { reset } from "redux-form";

export const collectionCreateActions = unionize({
    OPEN_COLLECTION_CREATOR: ofType<{ ownerUuid: string }>(),
    CLOSE_COLLECTION_CREATOR: ofType<{}>(),
    CREATE_COLLECTION: ofType<{}>(),
    CREATE_COLLECTION_SUCCESS: ofType<{}>(),
}, {
        tag: 'type',
        value: 'payload'
    });

export const createCollection = (collection: Partial<CollectionResource>, files: File[]) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { ownerUuid } = getState().collections.creator;
        const collectiontData = { ownerUuid, ...collection };
        dispatch(collectionCreateActions.CREATE_COLLECTION(collectiontData));
        const newCollection = await services.collectionService.create(collectiontData);
        await dispatch<any>(uploadCollectionFiles(newCollection.uuid));
        dispatch(collectionCreateActions.CREATE_COLLECTION_SUCCESS(collection));
        dispatch(reset('collectionCreateDialog'));
        return newCollection;
    };

export type CollectionCreateAction = UnionOf<typeof collectionCreateActions>;
