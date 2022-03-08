// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { dialogActions } from 'store/dialog/dialog-actions';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { fileUploaderActions } from 'store/file-uploader/file-uploader-actions';
import { reset, startSubmit, stopSubmit } from 'redux-form';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import * as WorkbenchActions from 'store/workbench/workbench-actions';

export const uploadCollectionFiles = (collectionUuid: string, targetLocation?: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(fileUploaderActions.START_UPLOAD());
        const files = getState().fileUploader.map(file => file.file);
        await services.collectionService.uploadFiles(collectionUuid, files, handleUploadProgress(dispatch), targetLocation);
        dispatch(WorkbenchActions.loadCollection(collectionUuid));
        dispatch(fileUploaderActions.CLEAR_UPLOAD());
    };

export const COLLECTION_UPLOAD_FILES_DIALOG = 'uploadCollectionFilesDialog';

export const openUploadCollectionFilesDialog = (targetLocation?: string) => (dispatch: Dispatch) => {
    dispatch(reset(COLLECTION_UPLOAD_FILES_DIALOG));
    dispatch(fileUploaderActions.CLEAR_UPLOAD());
    dispatch<any>(dialogActions.OPEN_DIALOG({ id: COLLECTION_UPLOAD_FILES_DIALOG, data: { targetLocation } }));
};

export const submitCollectionFiles = (targetLocation?: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            try {
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_UPLOAD_FILES_DIALOG));
                dispatch(startSubmit(COLLECTION_UPLOAD_FILES_DIALOG));
                await dispatch<any>(uploadCollectionFiles(currentCollection.uuid, targetLocation));
                dispatch(closeUploadCollectionFilesDialog());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Data has been uploaded.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_UPLOAD_FILES_DIALOG));
            } catch (e) {
                dispatch(stopSubmit(COLLECTION_UPLOAD_FILES_DIALOG));
                dispatch(closeUploadCollectionFilesDialog());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Data has not been uploaded. Too large file',
                    hideDuration: 2000,
                    kind: SnackbarKind.ERROR
                }));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_UPLOAD_FILES_DIALOG));
            }
        }
    };

export const closeUploadCollectionFilesDialog = () => dialogActions.CLOSE_DIALOG({ id: COLLECTION_UPLOAD_FILES_DIALOG });

const handleUploadProgress = (dispatch: Dispatch) => (fileId: number, loaded: number, total: number, currentTime: number) => {
    dispatch(fileUploaderActions.SET_UPLOAD_PROGRESS({ fileId, loaded, total, currentTime }));
};