// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { difference } from "lodash";
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

export const COLLECTION_PARTIAL_COPY_FORM_NAME = 'COLLECTION_PARTIAL_COPY_DIALOG';
export const COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION = 'COLLECTION_PARTIAL_COPY_TO_SELECTED_DIALOG';

export interface CollectionPartialCopyFormData {
    name: string;
    description: string;
    projectUuid: string;
}

export interface CollectionPartialCopyToSelectedCollectionFormData {
    collectionUuid: string;
}

export const openCollectionPartialCopyDialog = () =>
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

export const copyCollectionPartial = ({ name, description, projectUuid }: CollectionPartialCopyFormData) =>
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

export const openCollectionPartialCopyToSelectedCollectionDialog = () =>
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

export const copyCollectionPartialToSelectedCollection = ({ collectionUuid }: CollectionPartialCopyToSelectedCollectionFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
        const state = getState();
        const currentCollection = state.collectionPanel.item;

        if (currentCollection && !currentCollection.manifestText) {
            const fetchedCurrentCollection = await services.collectionService.get(currentCollection.uuid, undefined, ['manifestText']);
            currentCollection.manifestText = fetchedCurrentCollection.manifestText;
            currentCollection.unsignedManifestText = fetchedCurrentCollection.unsignedManifestText;
        }

        if (currentCollection) {
            try {
                dispatch(progressIndicatorActions.START_WORKING(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION));
                const selectedCollection = await services.collectionService.get(collectionUuid);
                const paths = filterCollectionFilesBySelection(state.collectionPanelFiles, false).map(file => file.id);
                const pathsToRemove = paths.filter(path => {
                    const a = path.split('/');
                    const fileExistsInSelectedCollection = selectedCollection.manifestText.includes(a[1]);
                    if (fileExistsInSelectedCollection) {
                        return path;
                    } else {
                        return null;
                    }
                });
                const diffPathToRemove = difference(paths, pathsToRemove);
                await services.collectionService.deleteFiles(selectedCollection.uuid, pathsToRemove.map(path => path.replace(currentCollection.uuid, collectionUuid)));
                const collectionWithDeletedFiles = await services.collectionService.get(collectionUuid, undefined, ['uuid', 'manifestText']);
                await services.collectionService.update(collectionUuid, { manifestText: `${collectionWithDeletedFiles.manifestText}${(currentCollection.manifestText ? currentCollection.manifestText : currentCollection.unsignedManifestText) || ''}` });
                await services.collectionService.deleteFiles(collectionWithDeletedFiles.uuid, diffPathToRemove.map(path => path.replace(currentCollection.uuid, collectionUuid)));
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