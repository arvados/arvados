// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { loadCollectionFiles } from '../../collection-panel/collection-panel-files/collection-panel-files-actions';
import { snackbarActions } from "~/store/snackbar/snackbar-actions";

export interface UploadFile {
    id: number;
    file: File;
    prevLoaded: number;
    loaded: number;
    total: number;
    startTime: number;
    prevTime: number;
    currentTime: number;
}

export const collectionUploaderActions = unionize({
    SET_UPLOAD_FILES: ofType<File[]>(),
    START_UPLOAD: ofType(),
    SET_UPLOAD_PROGRESS: ofType<{ fileId: number, loaded: number, total: number, currentTime: number }>(),
    CLEAR_UPLOAD: ofType()
}, {
        tag: 'type',
        value: 'payload'
    });

export type CollectionUploaderAction = UnionOf<typeof collectionUploaderActions>;

export const uploadCollectionFiles = (collectionUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(collectionUploaderActions.START_UPLOAD());
        const files = getState().collections.uploader.map(file => file.file);
        await services.collectionService.uploadFiles(collectionUuid, files, handleUploadProgress(dispatch));
        dispatch(collectionUploaderActions.CLEAR_UPLOAD());
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

export const UPLOAD_COLLECTION_FILES_DIALOG = 'uploadCollectionFilesDialog';
export const openUploadCollectionFilesDialog = () => (dispatch: Dispatch) => {
    dispatch(collectionUploaderActions.CLEAR_UPLOAD());
    dispatch<any>(dialogActions.OPEN_DIALOG({ id: UPLOAD_COLLECTION_FILES_DIALOG, data: {} }));
};

export const closeUploadCollectionFilesDialog = () => dialogActions.CLOSE_DIALOG({ id: UPLOAD_COLLECTION_FILES_DIALOG });

const handleUploadProgress = (dispatch: Dispatch) => (fileId: number, loaded: number, total: number, currentTime: number) => {
    dispatch(collectionUploaderActions.SET_UPLOAD_PROGRESS({ fileId, loaded, total, currentTime }));
};