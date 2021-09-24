// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { Dispatch } from "redux";
import servicesProvider from 'common/service-provider';
import { CollectionFilesTree, CollectionFileType, createCollectionFilesTree } from "models/collection-file";
import { ServiceRepository } from "services/services";
import { RootState } from "../../store";
import { snackbarActions, SnackbarKind } from "../../snackbar/snackbar-actions";
import { dialogActions } from '../../dialog/dialog-actions';
import { getNodeValue, mapTreeValues } from "models/tree";
import { filterCollectionFilesBySelection } from './collection-panel-files-state';
import { startSubmit, stopSubmit, initialize, FormErrors } from 'redux-form';
import { getDialog } from "store/dialog/dialog-reducer";
import { getFileFullPath, sortFilesTree } from "services/collection-service/collection-service-files-response";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { loadCollectionPanel } from "../collection-panel-action";

export const collectionPanelFilesAction = unionize({
    SET_COLLECTION_FILES: ofType<CollectionFilesTree>(),
    TOGGLE_COLLECTION_FILE_COLLAPSE: ofType<{ id: string }>(),
    TOGGLE_COLLECTION_FILE_SELECTION: ofType<{ id: string }>(),
    SELECT_ALL_COLLECTION_FILES: ofType<{}>(),
    UNSELECT_ALL_COLLECTION_FILES: ofType<{}>(),
    ON_SEARCH_CHANGE: ofType<string>(),
});

export type CollectionPanelFilesAction = UnionOf<typeof collectionPanelFilesAction>;

export const COLLECTION_PANEL_LOAD_FILES = 'collectionPanelLoadFiles';
export const COLLECTION_PANEL_LOAD_FILES_THRESHOLD = 40000;

export const setCollectionFiles = (files, joinParents = true) => (dispatch: any) => {
    const tree = createCollectionFilesTree(files, joinParents);
    const sorted = sortFilesTree(tree);
    const mapped = mapTreeValues(servicesProvider.getServices().collectionService.extendFileURL)(sorted);
    dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES(mapped));
};

export const loadCollectionFiles = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PANEL_LOAD_FILES));
        services.collectionService.files(uuid).then(files => {
            // Given the array of directories and files, create the appropriate tree nodes,
            // sort them, and add the complete url to each.
            const tree = createCollectionFilesTree(files);
            const sorted = sortFilesTree(tree);
            const mapped = mapTreeValues(services.collectionService.extendFileURL)(sorted);
            dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES(mapped));
            dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PANEL_LOAD_FILES));
        }).catch(() => {
            dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PANEL_LOAD_FILES));
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `Error getting file list`,
                hideDuration: 2000,
                kind: SnackbarKind.ERROR
            }));
        });
    };

export const removeCollectionFiles = (filePaths: string[]) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            services.collectionService.deleteFiles(currentCollection.uuid, filePaths).then(() => {
                dispatch<any>(loadCollectionPanel(currentCollection.uuid, true));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Removed.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
            }).catch(e =>
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Could not remove file.',
                    hideDuration: 2000,
                    kind: SnackbarKind.ERROR
                }))
            );
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
            const isDirectory = file.type === CollectionFileType.DIRECTORY;
            const title = isDirectory
                ? 'Removing directory'
                : 'Removing file';
            const text = isDirectory
                ? 'Are you sure you want to remove this directory?'
                : 'Are you sure you want to remove this file?';
            const info = isDirectory
                ? 'Removing files will change content address.'
                : 'Removing a file will change content address.';

            dispatch(dialogActions.OPEN_DIALOG({
                id: FILE_REMOVE_DIALOG,
                data: {
                    title,
                    text,
                    info,
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
            info: 'Removing files will change content address.',
            confirmButtonLabel: 'Remove'
        }
    });

export const RENAME_FILE_DIALOG = 'renameFileDialog';
export interface RenameFileDialogData {
    name: string;
    id: string;
    path: string;
}

export const openRenameFileDialog = (data: RenameFileDialogData) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(RENAME_FILE_DIALOG, data));
        dispatch(dialogActions.OPEN_DIALOG({ id: RENAME_FILE_DIALOG, data }));
    };

export const renameFile = (newFullPath: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const dialog = getDialog<RenameFileDialogData>(getState().dialog, RENAME_FILE_DIALOG);
        const currentCollection = getState().collectionPanel.item;
        if (dialog && currentCollection) {
            const file = getNodeValue(dialog.data.id)(getState().collectionPanelFiles);
            if (file) {
                dispatch(startSubmit(RENAME_FILE_DIALOG));
                const oldPath = getFileFullPath(file);
                const newPath = newFullPath;
                services.collectionService.moveFile(currentCollection.uuid, oldPath, newPath).then(() => {
                    dispatch<any>(loadCollectionPanel(currentCollection.uuid, true));
                    dispatch(dialogActions.CLOSE_DIALOG({ id: RENAME_FILE_DIALOG }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'File name changed.', hideDuration: 2000 }));
                }).catch(e => {
                    const errors: FormErrors<RenameFileDialogData, string> = {
                        path: `Could not rename the file: ${e.responseText}`
                    };
                    dispatch(stopSubmit(RENAME_FILE_DIALOG, errors));
                });
            }
        }
    };
