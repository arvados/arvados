// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { FormErrors, initialize, startSubmit, stopSubmit } from 'redux-form';
import { resetPickerProjectTree } from 'store/project-tree-picker/project-tree-picker-actions';
import { dialogActions } from 'store/dialog/dialog-actions';
import { ServiceRepository } from 'services/services';
import { filterCollectionFilesBySelection } from '../collection-panel/collection-panel-files/collection-panel-files-state';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getCommonResourceServiceError, CommonResourceServiceError } from 'services/common-service/common-resource-service';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { FileOperationLocation } from "store/tree-picker/tree-picker-actions";
import { updateResources } from 'store/resources/resources-actions';
import { navigateTo } from 'store/navigation/navigation-action';

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

export const openCollectionPartialCopyToNewCollectionDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            const initialData = {
                name: `Files extracted from: ${currentCollection.name}`,
                description: currentCollection.description,
                projectUuid: undefined
            };
            dispatch(initialize(COLLECTION_PARTIAL_COPY_FORM_NAME, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY_FORM_NAME, data: {} }));
        }
    };

export const copyCollectionPartialToNewCollection = ({ name, description, projectUuid }: CollectionPartialCopyToNewCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const state = getState();
        // Get current collection
        const sourceCollection = state.collectionPanel.item;

        if (sourceCollection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_COPY_FORM_NAME));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_FORM_NAME));

                // Get selected files
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, true)
                    .map(file => file.id.replace(new RegExp(`(^${sourceCollection.uuid})`), ''));

                // Copy files
                const updatedCollection = await services.collectionService.copyFiles(
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

export const openCollectionPartialCopyToExistingCollectionDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            const initialData = {
                destination: {uuid: currentCollection.uuid, destinationPath: ''}
            };
            dispatch(initialize(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION, data: {} }));
        }
    };

export const copyCollectionPartialToExistingCollection = ({ destination }: CollectionPartialCopyToExistingCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const state = getState();
        // Get current collection
        const sourceCollection = state.collectionPanel.item;

        if (sourceCollection && destination && destination.uuid) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
                // Get selected files
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, true)
                    .map(file => file.id.replace(new RegExp(`(^${sourceCollection.uuid})`), ''));

                // Copy files
                const updatedCollection = await services.collectionService.copyFiles(sourceCollection.portableDataHash, paths, {uuid: destination.uuid}, destination.path || '/', false);
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
        const currentCollection = getState().collectionPanel.item;
        if (currentCollection) {
            const initialData = {
                name: currentCollection.name,
                projectUuid: undefined
            };
            dispatch(initialize(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS, data: {} }));
        }
    };

export const copyCollectionPartialToSeparateCollections = ({ name, projectUuid }: CollectionPartialCopyToSeparateCollectionsFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const state = getState();
        // Get current collection
        const sourceCollection = state.collectionPanel.item;

        if (sourceCollection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS));

                // Get selected files
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, true)
                    .map(file => file.id.replace(new RegExp(`(^${sourceCollection.uuid})`), ''));

                // Copy files
                const collections = await Promise.all(paths.map((path) =>
                    services.collectionService.copyFiles(
                        sourceCollection.portableDataHash,
                        [path],
                        {
                            name: `File copied from collection ${name}${path}`,
                            ownerUuid: projectUuid,
                            uuid: undefined,
                        },
                        '/',
                        false
                    )
                ));
                dispatch(updateResources(collections));

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
