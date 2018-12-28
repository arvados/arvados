// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { startSubmit, stopSubmit, initialize, FormErrors } from 'redux-form';
import { ServiceRepository } from '~/services/services';
import { RootState } from '~/store/store';
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/services/common-service/common-resource-service";
import {snackbarActions, SnackbarKind} from '~/store/snackbar/snackbar-actions';
import { projectPanelActions } from '~/store/project-panel/project-panel-action';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';
import { resetPickerProjectTree } from '~/store/project-tree-picker/project-tree-picker-actions';
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { initProjectsTreePicker } from '~/store/tree-picker/tree-picker-actions';

export const COLLECTION_MOVE_FORM_NAME = 'collectionMoveFormName';

export const openMoveCollectionDialog = (resource: { name: string, uuid: string }) =>
    (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        dispatch<any>(initProjectsTreePicker(COLLECTION_MOVE_FORM_NAME));
        dispatch(initialize(COLLECTION_MOVE_FORM_NAME, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_MOVE_FORM_NAME, data: {} }));
    };

export const moveCollection = (resource: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(COLLECTION_MOVE_FORM_NAME));
        try {
            dispatch(progressIndicatorActions.START_WORKING(COLLECTION_MOVE_FORM_NAME));
            const collection = await services.collectionService.get(resource.uuid);
            await services.collectionService.update(resource.uuid, { ...collection, ownerUuid: resource.ownerUuid });
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_MOVE_FORM_NAME }));
            dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_MOVE_FORM_NAME));
            return collection;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(COLLECTION_MOVE_FORM_NAME, { ownerUuid: 'A collection with the same name already exists in the target project.' } as FormErrors));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_MOVE_FORM_NAME }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not move the collection.', hideDuration: 2000, kind: SnackbarKind.ERROR }));
            }
            dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_MOVE_FORM_NAME));
            return;
        }
    };
