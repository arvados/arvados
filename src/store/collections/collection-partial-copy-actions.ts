// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { FormErrors, initialize, startSubmit, stopSubmit } from 'redux-form';
import { resetPickerProjectTree } from '~/store/project-tree-picker/project-tree-picker-actions';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { ServiceRepository } from '~/services/services';
import { filterCollectionFilesBySelection } from '../collection-panel/collection-panel-files/collection-panel-files-state';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { getCommonResourceServiceError, CommonResourceServiceError } from '~/services/common-service/common-resource-service';
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { initProjectsTreePicker } from '~/store/tree-picker/tree-picker-actions';

export const COLLECTION_PARTIAL_COPY_FORM_NAME = 'COLLECTION_PARTIAL_COPY_DIALOG';

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
                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_PARTIAL_COPY_FORM_NAME }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'New collection created.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_PARTIAL_COPY_FORM_NAME));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
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
