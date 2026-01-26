// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
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

export const openCollectionCreateDialog = (ownerUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { router } = getState();
        let ownerUuidToUse = ownerUuid;
        if (!isProjectOrRunProcessRoute(router)) {
            const userUuid = getUserUuid(getState());
            if (!userUuid) { return; }
            ownerUuidToUse = userUuid;
        }
        dispatch(fileUploaderActions.CLEAR_UPLOAD());
        dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_CREATE_FORM_NAME, data: { ownerUuid: ownerUuidToUse } }));
    };

export const createCollection = (data: CollectionCreateFormDialogData, setSubmitErr: (errMsg: string) => void) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        let newCollection: CollectionResource | undefined;
        try {
            dispatch(progressIndicatorActions.START_WORKING(COLLECTION_CREATE_FORM_NAME));
            newCollection = await services.collectionService.create(data, false);
            await dispatch<any>(uploadCollectionFiles(newCollection.uuid));
            dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_CREATE_FORM_NAME }));
            return newCollection;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                setSubmitErr('Collection with the same name already exists.');
            } else {
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
