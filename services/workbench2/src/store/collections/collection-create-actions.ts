// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import {
    reset,
    startSubmit,
    stopSubmit,
    initialize,
    FormErrors,
    formValueSelector
} from 'redux-form';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { dialogActions } from "store/dialog/dialog-actions";
import { ServiceRepository } from 'services/services';
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { uploadCollectionFiles } from './collection-upload-actions';
import { fileUploaderActions } from 'store/file-uploader/file-uploader-actions';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { isProjectOrRunProcessRoute } from 'store/projects/project-create-actions';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { CollectionResource } from "models/collection";

export interface CollectionCreateFormDialogData {
    ownerUuid: string;
    name: string;
    description: string;
    storageClassesDesired: string[];
    properties: CollectionProperties;
}

export interface CollectionProperties {
    [key: string]: string | string[];
}

export const COLLECTION_CREATE_FORM_NAME = "collectionCreateFormName";
export const COLLECTION_CREATE_PROPERTIES_FORM_NAME = "collectionCreatePropertiesFormName";
export const COLLECTION_CREATE_FORM_SELECTOR = formValueSelector(COLLECTION_CREATE_FORM_NAME);

export const openCollectionCreateDialog = (ownerUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { router } = getState();
        if (!isProjectOrRunProcessRoute(router)) {
            const userUuid = getUserUuid(getState());
            if (!userUuid) { return; }
            dispatch(initialize(COLLECTION_CREATE_FORM_NAME, { ownerUuid: userUuid }));
        } else {
            dispatch(initialize(COLLECTION_CREATE_FORM_NAME, { ownerUuid }));
        }
        dispatch(fileUploaderActions.CLEAR_UPLOAD());
        dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_CREATE_FORM_NAME, data: { ownerUuid } }));
    };

export const createCollection = (data: CollectionCreateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(COLLECTION_CREATE_FORM_NAME));
        let newCollection: CollectionResource | undefined;
        try {
            dispatch(progressIndicatorActions.START_WORKING(COLLECTION_CREATE_FORM_NAME));
            newCollection = await services.collectionService.create(data, false);
            await dispatch<any>(uploadCollectionFiles(newCollection.uuid));
            dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_CREATE_FORM_NAME }));
            dispatch(reset(COLLECTION_CREATE_FORM_NAME));
            return newCollection;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(stopSubmit(COLLECTION_CREATE_FORM_NAME, { name: 'Collection with the same name already exists.' } as FormErrors));
            } else {
                dispatch(stopSubmit(COLLECTION_CREATE_FORM_NAME));
                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_CREATE_FORM_NAME }));
                const errMsg = e.errors
                    ? e.errors.join('')
                    : 'There was an error while creating the collection';
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: errMsg,
                    hideDuration: 2000,
                    kind: SnackbarKind.ERROR
                }));
                if (newCollection) { await services.collectionService.delete(newCollection.uuid); }
            }
            return;
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_CREATE_FORM_NAME));
        }
    };
