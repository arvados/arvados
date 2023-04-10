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
import { initProjectsTreePicker } from "store/tree-picker/tree-picker-actions";
import { updateResources } from 'store/resources/resources-actions';

export const COLLECTION_PARTIAL_COPY_FORM_NAME = 'COLLECTION_PARTIAL_COPY_DIALOG';
export const COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION = 'COLLECTION_PARTIAL_COPY_TO_SELECTED_DIALOG';

export interface CollectionPartialCopyToNewCollectionFormData {
    name: string;
    description: string;
    projectUuid: string;
}

export interface CollectionPartialCopyToExistingCollectionFormData {
    collectionUuid: string;
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
            dispatch<any>(initProjectsTreePicker(COLLECTION_PARTIAL_COPY_FORM_NAME));
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY_FORM_NAME, data: {} }));
        }
    };

export const copyCollectionPartialToNewCollection = ({ name, description, projectUuid }: CollectionPartialCopyToNewCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(COLLECTION_PARTIAL_COPY_FORM_NAME));
        const state = getState();
        const currentCollection = state.collectionPanel.item;
        if (currentCollection) {
            try {
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_FORM_NAME));
                const collectionManifestText = await services.collectionService.get(currentCollection.uuid, undefined, ['manifestText']);
                const collectionCopy = {
                    name,
                    description,
                    ownerUuid: projectUuid,
                    uuid: undefined,
                    manifestText: collectionManifestText.manifestText,
                };
                const newCollection = await services.collectionService.create(collectionCopy);
                const copiedFiles = await services.collectionService.files(newCollection.uuid);
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, true).map(file => file.id);
                const filesToDelete = copiedFiles.map(({ id }) => id).filter(file => {
                    return !paths.find(path => path.indexOf(file.replace(newCollection.uuid, '')) > -1);
                });
                await services.collectionService.deleteFiles(
                    newCollection.uuid,
                    filesToDelete
                );
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
                collectionUuid: ''
            };
            dispatch(initialize(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION, initialData));
            dispatch<any>(resetPickerProjectTree());
            dispatch<any>(initProjectsTreePicker(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
            dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION, data: {} }));
        }
    };

export const copyCollectionPartialToExistingCollection = ({ collectionUuid }: CollectionPartialCopyToExistingCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const state = getState();
        // Get current collection
        const sourceCollection = state.collectionPanel.item;

        if (sourceCollection) {
            try {
                dispatch(startSubmit(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
                // Get selected files
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, true)
                    .map(file => file.id.replace(new RegExp(`(^${sourceCollection.uuid})`), ''));

                // Copy files
                const updatedCollection = await services.collectionService.copyFiles(sourceCollection.portableDataHash, paths, collectionUuid, '/', false);
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
