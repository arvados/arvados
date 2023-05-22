// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { initialize, startSubmit } from "redux-form";
import { CommonResourceServiceError, getCommonResourceServiceError } from "services/common-service/common-resource-service";
import { ServiceRepository } from "services/services";
import { filterCollectionFilesBySelection } from "store/collection-panel/collection-panel-files/collection-panel-files-state";
import { dialogActions } from "store/dialog/dialog-actions";
import { navigateTo } from "store/navigation/navigation-action";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { resetPickerProjectTree } from "store/project-tree-picker/project-tree-picker-actions";
import { updateResources } from "store/resources/resources-actions";
import { SnackbarKind, snackbarActions } from "store/snackbar/snackbar-actions";
import { RootState } from "store/store";
import { FileOperationLocation } from "store/tree-picker/tree-picker-actions";

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

export const openCollectionPartialMoveToNewCollectionDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            const initialData = {
                name: `Files moved from: ${currentCollection.name}`,
                description: currentCollection.description,
                projectUuid: undefined
            };
            dispatch(initialize(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION, data: {} }));
        }
    };

export const moveCollectionPartialToNewCollection = ({ name, description, projectUuid }: CollectionPartialMoveToNewCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const state = getState();
        // Get current collection
        const sourceCollection = state.collectionPanel.item;

        if (sourceCollection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION));

                // Get selected files
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, true)
                    .map(file => file.id.replace(new RegExp(`(^${sourceCollection.uuid})`), ''));

                // Move files
                const updatedCollection = await services.collectionService.moveFiles(
                    sourceCollection.uuid,
                    sourceCollection.portableDataHash,
                    paths,
                    {
                        name,
                        description,
                        ownerUuid: projectUuid,
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
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not move files to selected collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                }
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION));
            }
        }
    };

export const openCollectionPartialMoveToExistingCollectionDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            const initialData = {
                destination: {uuid: currentCollection.uuid, path: ''}
            };
            dispatch(initialize(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION, data: {} }));
        }
    };

export const moveCollectionPartialToExistingCollection = ({ destination }: CollectionPartialMoveToExistingCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const state = getState();
        // Get current collection
        const sourceCollection = state.collectionPanel.item;

        if (sourceCollection && destination && destination.uuid) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION));
                // Get selected files
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, true)
                    .map(file => file.id.replace(new RegExp(`(^${sourceCollection.uuid})`), ''));

                // Move files
                const updatedCollection = await services.collectionService.moveFiles(sourceCollection.uuid, sourceCollection.portableDataHash, paths, {uuid: destination.uuid}, destination.path || '/', false);
                dispatch(updateResources([updatedCollection]));

                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Files have been moved to selected collection.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNKNOWN) {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not copy this files to selected collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
                }
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION));
            }
        }
    };

export const openCollectionPartialMoveToSeparateCollectionsDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            const initialData = {
                name: currentCollection.name,
                projectUuid: undefined
            };
            dispatch(initialize(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS, data: {} }));
        }
    };

export const moveCollectionPartialToSeparateCollections = ({ name, projectUuid }: CollectionPartialMoveToSeparateCollectionsFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const state = getState();
        // Get current collection
        const sourceCollection = state.collectionPanel.item;

        if (sourceCollection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS));

                // Get selected files
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, true)
                    .map(file => file.id.replace(new RegExp(`(^${sourceCollection.uuid})`), ''));

                // Move files
                const collections = await Promise.all(paths.map((path) =>
                    services.collectionService.moveFiles(
                        sourceCollection.uuid,
                        sourceCollection.portableDataHash,
                        [path],
                        {
                            name: `File moved from collection ${name}${path}`,
                            ownerUuid: projectUuid,
                            uuid: undefined,
                        },
                        '/',
                        false
                    )
                ));
                dispatch(updateResources(collections));
                dispatch<any>(navigateTo(projectUuid));

                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'New collections created.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS));
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
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS));
            }
        }
    };
