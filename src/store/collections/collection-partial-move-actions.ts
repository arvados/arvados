// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { initialize, startSubmit, stopSubmit } from "redux-form";
import { CommonResourceServiceError, getCommonResourceServiceError } from "services/common-service/common-resource-service";
import { ServiceRepository } from "services/services";
import { CollectionFileSelection, CollectionPanelDirectory, CollectionPanelFile, filterCollectionFilesBySelection, getCollectionSelection } from "store/collection-panel/collection-panel-files/collection-panel-files-state";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { dialogActions } from "store/dialog/dialog-actions";
import { navigateTo } from "store/navigation/navigation-action";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { resetPickerProjectTree } from "store/project-tree-picker/project-tree-picker-actions";
import { updateResources } from "store/resources/resources-actions";
import { SnackbarKind, snackbarActions } from "store/snackbar/snackbar-actions";
import { RootState } from "store/store";
import { FileOperationLocation } from "store/tree-picker/tree-picker-actions";
import { CollectionResource } from "models/collection";
import { SOURCE_DESTINATION_EQUAL_ERROR_MESSAGE } from "services/collection-service/collection-service";

export const COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION = 'COLLECTION_PARTIAL_MOVE_TO_NEW_DIALOG';
export const COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION = 'COLLECTION_PARTIAL_MOVE_TO_SELECTED_DIALOG';
export const COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS = 'COLLECTION_PARTIAL_MOVE_TO_SEPARATE_DIALOG';

export interface CollectionPartialMoveToNewCollectionFormData {
    name: string;
    description: string;
    projectUuid: string;
}

export interface CollectionPartialMoveToExistingCollectionFormData {
    destination: FileOperationLocation;
}

export interface CollectionPartialMoveToSeparateCollectionsFormData {
    name: string;
    projectUuid: string;
}

export const openCollectionPartialMoveToNewCollectionDialog = (resource: ContextMenuResource) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;

        if (sourceCollection) {
            openMoveToNewDialog(dispatch, sourceCollection, [resource]);
        }
    };

export const openCollectionPartialMoveMultipleToNewCollectionDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;
        const selectedItems = filterCollectionFilesBySelection(getState().collectionPanelFiles, true);

        if (sourceCollection && selectedItems.length) {
            openMoveToNewDialog(dispatch, sourceCollection, selectedItems);
        }
    };

const openMoveToNewDialog = (dispatch: Dispatch, sourceCollection: CollectionResource, selectedItems: (CollectionPanelDirectory | CollectionPanelFile | ContextMenuResource)[]) => {
    // Get selected files
    const collectionFileSelection = getCollectionSelection(sourceCollection, selectedItems);
    // Populate form initial state
    const initialFormData = {
        name: `Files moved from: ${sourceCollection.name}`,
        description: sourceCollection.description,
        projectUuid: undefined
    };
    dispatch(initialize(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION, initialFormData));
    dispatch<any>(resetPickerProjectTree());
    dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION, data: collectionFileSelection }));
}

export const moveCollectionPartialToNewCollection = (fileSelection: CollectionFileSelection, formData: CollectionPartialMoveToNewCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (fileSelection.collection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION));

                // Move files
                const updatedCollection = await services.collectionService.moveFiles(
                    fileSelection.collection.uuid,
                    fileSelection.collection.portableDataHash,
                    fileSelection.selectedPaths,
                    {
                        name: formData.name,
                        description: formData.description,
                        ownerUuid: formData.projectUuid,
                        uuid: undefined,
                    },
                    '/',
                    false
                );
                dispatch(updateResources([updatedCollection]));
                dispatch<any>(navigateTo(updatedCollection.uuid));

                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Files have been moved to selected collection.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not move files to selected collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                }
            } finally {
                dispatch(stopSubmit(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION));
            }
        }
    };

export const openCollectionPartialMoveToExistingCollectionDialog = (resource: ContextMenuResource) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;

        if (sourceCollection) {
            openMoveToExistingDialog(dispatch, sourceCollection, [resource]);
        }
    };

export const openCollectionPartialMoveMultipleToExistingCollectionDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;
        const selectedItems = filterCollectionFilesBySelection(getState().collectionPanelFiles, true);

        if (sourceCollection && selectedItems.length) {
            openMoveToExistingDialog(dispatch, sourceCollection, selectedItems);
        }
    };

const openMoveToExistingDialog = (dispatch: Dispatch, sourceCollection: CollectionResource, selectedItems: (CollectionPanelDirectory | CollectionPanelFile | ContextMenuResource)[]) => {
    // Get selected files
    const collectionFileSelection = getCollectionSelection(sourceCollection, selectedItems);
    // Populate form initial state
    const initialFormData = {
        destination: {uuid: sourceCollection.uuid, path: ''}
    };
    dispatch(initialize(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION, initialFormData));
    dispatch<any>(resetPickerProjectTree());
    dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION, data: collectionFileSelection }));
}

export const moveCollectionPartialToExistingCollection = (fileSelection: CollectionFileSelection, formData: CollectionPartialMoveToExistingCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (fileSelection.collection && formData.destination && formData.destination.uuid) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION));

                // Move files
                const updatedCollection = await services.collectionService.moveFiles(
                    fileSelection.collection.uuid,
                    fileSelection.collection.portableDataHash,
                    fileSelection.selectedPaths,
                    {uuid: formData.destination.uuid},
                    formData.destination.subpath || '/', false
                );
                dispatch(updateResources([updatedCollection]));

                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Files have been moved to selected collection.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.SOURCE_DESTINATION_CANNOT_BE_SAME) {
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: SOURCE_DESTINATION_EQUAL_ERROR_MESSAGE, hideDuration: 2000, kind: SnackbarKind.ERROR }));
                } else if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not copy this files to selected collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                }
            } finally {
                dispatch(stopSubmit(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION));
            }
        }
    };

export const openCollectionPartialMoveToSeparateCollectionsDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;
        const selectedItems = filterCollectionFilesBySelection(getState().collectionPanelFiles, true);

        if (sourceCollection && selectedItems.length) {
            // Get selected files
            const collectionFileSelection = getCollectionSelection(sourceCollection, selectedItems);
            // Populate form initial state
            const initialData = {
                name: sourceCollection.name,
                projectUuid: undefined
            };
            dispatch(initialize(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS, data: collectionFileSelection }));
        }
    };

export const moveCollectionPartialToSeparateCollections = (fileSelection: CollectionFileSelection, formData: CollectionPartialMoveToSeparateCollectionsFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (fileSelection.collection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS));

                // Move files
                const collections = await Promise.all(fileSelection.selectedPaths.map((path) =>
                    services.collectionService.moveFiles(
                        fileSelection.collection.uuid,
                        fileSelection.collection.portableDataHash,
                        [path],
                        {
                            name: `File moved from collection ${formData.name}${path}`,
                            ownerUuid: formData.projectUuid,
                            uuid: undefined,
                        },
                        '/',
                        false
                    )
                ));
                dispatch(updateResources(collections));
                dispatch<any>(navigateTo(formData.projectUuid));

                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'New collections created.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection from one or more files already exists', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                } else if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not create a copy of collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                } else {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection has been copied but may contain incorrect files.', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                }
            } finally {
                dispatch(stopSubmit(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS));
            }
        }
    };
