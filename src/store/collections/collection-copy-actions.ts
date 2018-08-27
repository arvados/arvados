// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { initialize, startSubmit, stopSubmit } from 'redux-form';
import { resetPickerProjectTree } from '~/store/project-tree-picker/project-tree-picker-actions';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { getCommonResourceServiceError, CommonResourceServiceError } from '~/common/api/common-resource-service';

export const COLLECTION_COPY_FORM_NAME = 'collectionCopyFormName';

export interface CollectionCopyFormDialogData {
    name: string;
    ownerUuid: string;
    uuid: string;
}

export const openCollectionCopyDialog = (resource: { name: string, uuid: string }) =>
    (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        const initialData: CollectionCopyFormDialogData = { name: `Copy of: ${resource.name}`, ownerUuid: '', uuid: resource.uuid };
        dispatch<any>(initialize(COLLECTION_COPY_FORM_NAME, initialData));
        dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_COPY_FORM_NAME, data: {} }));
    };

export const copyCollection = (resource: CollectionCopyFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(COLLECTION_COPY_FORM_NAME));
        try {
            const collection = await services.collectionService.get(resource.uuid);
            const uuidKey = 'uuid';
            delete collection[uuidKey];
            await services.collectionService.create({ ...collection, ownerUuid: resource.ownerUuid, name: resource.name });
            dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_COPY_FORM_NAME }));
            return collection;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(COLLECTION_COPY_FORM_NAME, { ownerUuid: 'A collection with the same name already exists in the target project.' }));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_COPY_FORM_NAME }));
                throw new Error('Could not copy the collection.');
            }
            return ;
        }
    };