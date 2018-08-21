// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { CollectionFilesTree, CollectionFileType } from "~/models/collection-file";
import { ServiceRepository } from "~/services/services";
import { RootState } from "../../store";
import { snackbarActions } from "../../snackbar/snackbar-actions";
import { dialogActions } from '../../dialog/dialog-actions';
import { getNodeValue } from "~/models/tree";
import { filterCollectionFilesBySelection } from './collection-panel-files-state';
import { startSubmit, initialize, stopSubmit, reset } from 'redux-form';
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/common/api/common-resource-service";
import { resetPickerProjectTree } from '../../project-tree-picker/project-tree-picker-actions';
import { getDialog } from "~/store/dialog/dialog-reducer";

export const collectionPanelFilesAction = unionize({
    SET_COLLECTION_FILES: ofType<CollectionFilesTree>(),
    TOGGLE_COLLECTION_FILE_COLLAPSE: ofType<{ id: string }>(),
    TOGGLE_COLLECTION_FILE_SELECTION: ofType<{ id: string }>(),
    SELECT_ALL_COLLECTION_FILES: ofType<{}>(),
    UNSELECT_ALL_COLLECTION_FILES: ofType<{}>(),
}, { tag: 'type', value: 'payload' });

export type CollectionPanelFilesAction = UnionOf<typeof collectionPanelFilesAction>;

export const loadCollectionFiles = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const files = await services.collectionService.files(uuid);
        dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES(files));
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
        const paths = filterCollectionFilesBySelection(getState().collectionPanelFiles, true).map(file => file.id);
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

export const COLLECTION_PARTIAL_COPY = 'COLLECTION_PARTIAL_COPY';

export interface CollectionPartialCopyFormData {
    name: string;
    description: string;
    projectUuid: string;
}

export const openCollectionPartialCopyDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            const initialData = {
                name: currentCollection.name,
                description: currentCollection.description,
                projectUuid: ''
            };
            dispatch(initialize(COLLECTION_PARTIAL_COPY, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY, data: {} }));
        }
    };

export const doCollectionPartialCopy = ({ name, description, projectUuid }: CollectionPartialCopyFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(COLLECTION_PARTIAL_COPY));
        const state = getState();
        const currentCollection = state.collectionPanel.item;
        if (currentCollection) {
            try {
                const collection = await services.collectionService.get(currentCollection.uuid);
                const collectionCopy = {
                    ...collection,
                    name,
                    description,
                    ownerUuid: projectUuid,
                    uuid: undefined
                };
                const newCollection = await services.collectionService.create(collectionCopy);
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, false).map(file => file.id);
                await services.collectionService.deleteFiles(newCollection.uuid, paths);
                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'New collection created.', hideDuration: 2000 }));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                    dispatch(stopSubmit(COLLECTION_PARTIAL_COPY, { name: 'Collection with this name already exists.' }));
                } else if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not create a copy of collection', hideDuration: 2000 }));
                } else {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection has been copied but may contain incorrect files.', hideDuration: 2000 }));
                }
            }
        }
    };

export const RENAME_FILE_DIALOG = 'renameFileDialog';
export interface RenameFileDialogData {
    name: string;
    id: string;
}

export const openRenameFileDialog = (data: RenameFileDialogData) =>
    (dispatch: Dispatch) => {
        dispatch(reset(RENAME_FILE_DIALOG));
        dispatch(dialogActions.OPEN_DIALOG({ id: RENAME_FILE_DIALOG, data }));
    };

export const renameFile = (newName: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const dialog = getDialog<RenameFileDialogData>(getState().dialog, RENAME_FILE_DIALOG);
        const currentCollection = getState().collectionPanel.item;
        if (dialog && currentCollection) {
            dispatch(startSubmit(RENAME_FILE_DIALOG));
            const oldPath = dialog.data.id;
            const newPath = dialog.data.id.replace(dialog.data.name, newName);
            try {
                await services.collectionService.moveFile(currentCollection.uuid, oldPath, newPath);
                dispatch<any>(loadCollectionFiles(currentCollection.uuid));
                dispatch(dialogActions.CLOSE_DIALOG({ id: RENAME_FILE_DIALOG }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'File name changed.', hideDuration: 2000 }));
            } catch (e) {
                dispatch(stopSubmit(RENAME_FILE_DIALOG, { name: 'Could not rename the file' }));
            }
        }
    };
