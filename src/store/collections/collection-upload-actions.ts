// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { loadCollectionFiles } from '../collection-panel/collection-panel-files/collection-panel-files-actions';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { fileUploaderActions } from '~/store/file-uploader/file-uploader-actions';
import { reset } from 'redux-form';

export const uploadCollectionFiles = (collectionUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(fileUploaderActions.START_UPLOAD());
        const files = getState().fileUploader.map(file => file.file);
        await services.collectionService.uploadFiles(collectionUuid, files, handleUploadProgress(dispatch));
        dispatch(fileUploaderActions.CLEAR_UPLOAD());
    };

export const UPLOAD_COLLECTION_FILES_DIALOG = 'uploadCollectionFilesDialog';

export const openUploadCollectionFilesDialog = () => (dispatch: Dispatch) => {
    dispatch(reset(UPLOAD_COLLECTION_FILES_DIALOG));
    dispatch(fileUploaderActions.CLEAR_UPLOAD());
    dispatch<any>(dialogActions.OPEN_DIALOG({ id: UPLOAD_COLLECTION_FILES_DIALOG, data: {} }));
};

export const uploadCurrentCollectionFiles = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            await dispatch<any>(uploadCollectionFiles(currentCollection.uuid));
            dispatch<any>(loadCollectionFiles(currentCollection.uuid));
            dispatch(closeUploadCollectionFilesDialog());
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Data has been uploaded.', hideDuration: 2000 }));
        }
    };

export const closeUploadCollectionFilesDialog = () => dialogActions.CLOSE_DIALOG({ id: UPLOAD_COLLECTION_FILES_DIALOG });

const handleUploadProgress = (dispatch: Dispatch) => (fileId: number, loaded: number, total: number, currentTime: number) => {
    dispatch(fileUploaderActions.SET_UPLOAD_PROGRESS({ fileId, loaded, total, currentTime }));
};