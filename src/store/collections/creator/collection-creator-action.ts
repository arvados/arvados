// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";

import { RootState } from "../../store";
import { CollectionResource } from '../../../models/collection';
import { ServiceRepository } from "../../../services/services";
import { collectionUploaderActions } from "../uploader/collection-uploader-actions";
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
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { ownerUuid } = getState().collections.creator;
        const collectiontData = { ownerUuid, ...collection };
        dispatch(collectionCreateActions.CREATE_COLLECTION(collectiontData));
        return services.collectionService
            .create(collectiontData)
            .then(collection => {
                dispatch(collectionUploaderActions.START_UPLOAD());
                services.collectionService.uploadFiles(collection.uuid, files,
                    (fileId, loaded, total, currentTime) => {
                        dispatch(collectionUploaderActions.SET_UPLOAD_PROGRESS({ fileId, loaded, total, currentTime }));
                    })
                    .then(() => {
                        dispatch(collectionCreateActions.CREATE_COLLECTION_SUCCESS(collection));
                        dispatch(reset('collectionCreateDialog'));
                        dispatch(collectionUploaderActions.CLEAR_UPLOAD());
                    });
                return collection;
            });
    };

export type CollectionCreateAction = UnionOf<typeof collectionCreateActions>;
