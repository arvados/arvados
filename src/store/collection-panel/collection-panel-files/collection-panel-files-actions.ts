// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { CollectionFilesTree, CollectionFileType } from "../../../models/collection-file";
import { ServiceRepository } from "../../../services/services";
import { RootState } from "../../store";
import { snackbarActions } from "../../snackbar/snackbar-actions";
import { dialogActions } from "../../dialog/dialog-actions";
import { getNodeValue } from "../../../models/tree";

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
        const { item } = getState().collectionPanel;
        if (item) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...' }));
            const promises = filePaths.map(filePath => services.collectionService.deleteFile(item.uuid, filePath));
            await Promise.all(promises);
            dispatch<any>(loadCollectionFiles(item.uuid));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000 }));
        }
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