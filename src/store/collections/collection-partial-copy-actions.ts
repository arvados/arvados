// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { FormErrors, initialize, startSubmit, stopSubmit } from 'redux-form';
import { resetPickerProjectTree } from 'store/project-tree-picker/project-tree-picker-actions';
import { dialogActions } from 'store/dialog/dialog-actions';
import { ServiceRepository } from 'services/services';
import { CollectionFileSelection, CollectionPanelDirectory, CollectionPanelFile, filterCollectionFilesBySelection, getCollectionSelection } from '../collection-panel/collection-panel-files/collection-panel-files-state';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getCommonResourceServiceError, CommonResourceServiceError } from 'services/common-service/common-resource-service';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { FileOperationLocation } from "store/tree-picker/tree-picker-actions";
import { updateResources } from 'store/resources/resources-actions';
import { navigateTo } from 'store/navigation/navigation-action';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { CollectionResource } from 'models/collection';

export const COLLECTION_PARTIAL_COPY_FORM_NAME = 'COLLECTION_PARTIAL_COPY_DIALOG';
export const COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION = 'COLLECTION_PARTIAL_COPY_TO_SELECTED_DIALOG';
export const COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS = 'COLLECTION_PARTIAL_COPY_TO_SEPARATE_DIALOG';

export interface CollectionPartialCopyToNewCollectionFormData {
    name: string;
    description: string;
    projectUuid: string;
}

export interface CollectionPartialCopyToExistingCollectionFormData {
    destination: FileOperationLocation;
}

export interface CollectionPartialCopyToSeparateCollectionsFormData {
    name: string;
    projectUuid: string;
}

export const openCollectionPartialCopyToNewCollectionDialog = (resource: ContextMenuResource) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;

        if (sourceCollection) {
            openCopyToNewDialog(dispatch, sourceCollection, [resource]);
        }
    };

export const openCollectionPartialCopyMultipleToNewCollectionDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;
        const selectedItems = filterCollectionFilesBySelection(getState().collectionPanelFiles, true);

        if (sourceCollection && selectedItems.length) {
            openCopyToNewDialog(dispatch, sourceCollection, selectedItems);
        }
    };

const openCopyToNewDialog = (dispatch: Dispatch, sourceCollection: CollectionResource, selectedItems: (CollectionPanelDirectory | CollectionPanelFile | ContextMenuResource)[]) => {
    // Get selected files
    const collectionFileSelection = getCollectionSelection(sourceCollection, selectedItems);
    // Populate form initial state
    const initialFormData = {
        name: `Files extracted from: ${sourceCollection.name}`,
        description: sourceCollection.description,
        projectUuid: undefined
    };
    dispatch(initialize(COLLECTION_PARTIAL_COPY_FORM_NAME, initialFormData));
    dispatch<any>(resetPickerProjectTree());
    dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY_FORM_NAME, data: collectionFileSelection }));
};

export const copyCollectionPartialToNewCollection = (fileSelection: CollectionFileSelection, formData: CollectionPartialCopyToNewCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (fileSelection.collection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_COPY_FORM_NAME));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_FORM_NAME));

                // Copy files
                const updatedCollection = await services.collectionService.copyFiles(
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

                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_FORM_NAME }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'New collection created.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_COPY_FORM_NAME));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                    dispatch(stopSubmit(COLLECTION_PARTIAL_COPY_FORM_NAME, { name: 'Collection with this name already exists.' } as FormErrors));
                } else if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_FORM_NAME }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not create a copy of collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                } else {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_FORM_NAME }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection has been copied but may contain incorrect files.', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                }
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_COPY_FORM_NAME));
            }
        }
    };

export const openCollectionPartialCopyToExistingCollectionDialog = (resource: ContextMenuResource) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;

        if (sourceCollection) {
            openCopyToExistingDialog(dispatch, sourceCollection, [resource]);
        }
    };

export const openCollectionPartialCopyMultipleToExistingCollectionDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;
        const selectedItems = filterCollectionFilesBySelection(getState().collectionPanelFiles, true);

        if (sourceCollection && selectedItems.length) {
            openCopyToExistingDialog(dispatch, sourceCollection, selectedItems);
        }
    };

const openCopyToExistingDialog = (dispatch: Dispatch, sourceCollection: CollectionResource, selectedItems: (CollectionPanelDirectory | CollectionPanelFile | ContextMenuResource)[]) => {
    // Get selected files
    const collectionFileSelection = getCollectionSelection(sourceCollection, selectedItems);
    // Populate form initial state
    const initialFormData = {
        destination: {uuid: sourceCollection.uuid, destinationPath: ''}
    };
    dispatch(initialize(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION, initialFormData));
    dispatch<any>(resetPickerProjectTree());
    dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION, data: collectionFileSelection }));
}

export const copyCollectionPartialToExistingCollection = (fileSelection: CollectionFileSelection, formData: CollectionPartialCopyToExistingCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (fileSelection.collection && formData.destination && formData.destination.uuid) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));

                // Copy files
                const updatedCollection = await services.collectionService.copyFiles(fileSelection.collection.portableDataHash, fileSelection.selectedPaths, {uuid: formData.destination.uuid}, formData.destination.path || '/', false);
                dispatch(updateResources([updatedCollection]));

                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Files has been copied to selected collection.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not copy this files to selected collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                }
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
            }
        }
    };

export const openCollectionPartialCopyToSeparateCollectionsDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sourceCollection = getState().collectionPanel.item;
        const selectedItems = filterCollectionFilesBySelection(getState().collectionPanelFiles, true);

        if (sourceCollection && selectedItems.length) {
            // Get selected files
            const collectionFileSelection = getCollectionSelection(sourceCollection, selectedItems);
            // Populate form initial state
            const initialFormData = {
                name: sourceCollection.name,
                projectUuid: undefined
            };
            dispatch(initialize(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS, initialFormData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS, data: collectionFileSelection }));
        }
    };

export const copyCollectionPartialToSeparateCollections = (fileSelection: CollectionFileSelection, formData: CollectionPartialCopyToSeparateCollectionsFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (fileSelection.collection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS));

                // Copy files
                const collections = await Promise.all(fileSelection.selectedPaths.map((path) =>
                    services.collectionService.copyFiles(
                        fileSelection.collection.portableDataHash,
                        [path],
                        {
                            name: `File copied from collection ${formData.name}${path}`,
                            ownerUuid: formData.projectUuid,
                            uuid: undefined,
                        },
                        '/',
                        false
                    )
                ));
                dispatch(updateResources(collections));
                dispatch<any>(navigateTo(formData.projectUuid));

                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'New collections created.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                console.log(e, error);
                if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection from one or more files already exists', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                } else if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not create a copy of collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                } else {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection has been copied but may contain incorrect files.', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                }
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS));
            }
        }
    };
