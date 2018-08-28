// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { reset, startSubmit, stopSubmit, initialize } from 'redux-form';
import { RootState } from '~/store/store';
import { uploadCollectionFiles } from '~/store/collections/uploader/collection-uploader-actions';
import { dialogActions } from "~/store/dialog/dialog-actions";
import { CollectionResource } from '~/models/collection';
import { ServiceRepository } from '~/services/services';
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/common/api/common-resource-service";

export interface CollectionCreateFormDialogData {
    ownerUuid: string;
    name: string;
    description: string;
    files: File[];
}

export const COLLECTION_CREATE_FORM_NAME = "collectionCreateFormName";

export const openCollectionCreateDialog = (ownerUuid: string) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(COLLECTION_CREATE_FORM_NAME, { ownerUuid }));
        dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_CREATE_FORM_NAME, data: { ownerUuid } }));
    };

export const createCollection = (collection: Partial<CollectionResource>) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(COLLECTION_CREATE_FORM_NAME));
        try {
            const newCollection = await services.collectionService.create(collection);
            await dispatch<any>(uploadCollectionFiles(newCollection.uuid));
            dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_CREATE_FORM_NAME }));
            dispatch(reset(COLLECTION_CREATE_FORM_NAME));
            return newCollection;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(COLLECTION_CREATE_FORM_NAME, { name: 'Collection with the same name already exists.' }));
            }
            return ;
        }
    };