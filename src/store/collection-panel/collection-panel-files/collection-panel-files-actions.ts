// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";
import { Dispatch } from "redux";
import { CollectionFilesTree, CollectionFileType } from "~/models/collection-file";
import { ServiceRepository } from "~/services/services";
import { RootState } from "../../store";
import { snackbarActions } from "../../snackbar/snackbar-actions";
import { dialogActions } from '../../dialog/dialog-actions';
import { getNodeValue } from "~/models/tree";
import { filterCollectionFilesBySelection } from './collection-panel-files-state';
import { startSubmit, stopSubmit, reset, initialize } from 'redux-form';
import { getDialog } from "~/store/dialog/dialog-reducer";
import { getFileFullPath } from "~/services/collection-service/collection-service-files-response";
import { resourcesDataActions } from "~/store/resources-data/resources-data-actions";

export const collectionPanelFilesAction = unionize({
    SET_COLLECTION_FILES: ofType<CollectionFilesTree>(),
    TOGGLE_COLLECTION_FILE_COLLAPSE: ofType<{ id: string }>(),
    TOGGLE_COLLECTION_FILE_SELECTION: ofType<{ id: string }>(),
    SELECT_ALL_COLLECTION_FILES: ofType<{}>(),
    UNSELECT_ALL_COLLECTION_FILES: ofType<{}>(),
});

export type CollectionPanelFilesAction = UnionOf<typeof collectionPanelFilesAction>;

export const loadCollectionFiles = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const files = await services.collectionService.files(uuid);
        dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES(files));
        dispatch(resourcesDataActions.SET_FILES({ uuid, files }));
    };

export const removeCollectionFiles = (filePaths: string[]) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...' }));
            await services.collectionService.deleteFiles(currentCollection.uuid, filePaths);
            dispatch<any>(loadCollectionFiles(currentCollection.uuid));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000 }));
        }
    };

export const removeCollectionsSelectedFiles = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const paths = filterCollectionFilesBySelection(getState().collectionPanelFiles, true)
            .map(getFileFullPath);
        dispatch<any>(removeCollectionFiles(paths));
    };

export const FILE_REMOVE_DIALOG = 'fileRemoveDialog';

export const openFileRemoveDialog = (filePath: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const file = getNodeValue(filePath)(getState().collectionPanelFiles);
        if (file) {
            const title = file.type === CollectionFileType.DIRECTORY
                ? 'Removing directory'
                : 'Removing file';
            const text = file.type === CollectionFileType.DIRECTORY
                ? 'Are you sure you want to remove this directory?'
                : 'Are you sure you want to remove this file?';

            dispatch(dialogActions.OPEN_DIALOG({
                id: FILE_REMOVE_DIALOG,
                data: {
                    title,
                    text,
                    confirmButtonLabel: 'Remove',
                    filePath
                }
            }));
        }
    };

export const MULTIPLE_FILES_REMOVE_DIALOG = 'multipleFilesRemoveDialog';

export const openMultipleFilesRemoveDialog = () =>
    dialogActions.OPEN_DIALOG({
        id: MULTIPLE_FILES_REMOVE_DIALOG,
        data: {
            title: 'Removing files',
            text: 'Are you sure you want to remove selected files?',
            confirmButtonLabel: 'Remove'
        }
    });

export const RENAME_FILE_DIALOG = 'renameFileDialog';
export interface RenameFileDialogData {
    name: string;
    id: string;
}

export const openRenameFileDialog = (data: RenameFileDialogData) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(RENAME_FILE_DIALOG, data));
        dispatch(dialogActions.OPEN_DIALOG({ id: RENAME_FILE_DIALOG, data }));
    };

export const renameFile = (newName: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const dialog = getDialog<RenameFileDialogData>(getState().dialog, RENAME_FILE_DIALOG);
        const currentCollection = getState().collectionPanel.item;
        if (dialog && currentCollection) {
            const file = getNodeValue(dialog.data.id)(getState().collectionPanelFiles);
            if (file) {
                dispatch(startSubmit(RENAME_FILE_DIALOG));
                const oldPath = getFileFullPath(file);
                const newPath = getFileFullPath({ ...file, name: newName });
                try {
                    await services.collectionService.moveFile(currentCollection.uuid, oldPath, newPath);
                    dispatch<any>(loadCollectionFiles(currentCollection.uuid));
                    dispatch(dialogActions.CLOSE_DIALOG({ id: RENAME_FILE_DIALOG }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'File name changed.', hideDuration: 2000 }));
                } catch (e) {
                    dispatch(stopSubmit(RENAME_FILE_DIALOG, { name: 'Could not rename the file' }));
                }
            }
        }
    };
