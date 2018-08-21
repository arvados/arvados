// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { reduxForm, startSubmit, stopSubmit, initialize } from 'redux-form';
import { ServiceRepository } from '~/services/services';
import { RootState } from '~/store/store';
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/common/api/common-resource-service";
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { MoveToFormDialogData, MoveToFormDialog } from '../move-to-dialog/move-to-dialog';
import { projectPanelActions } from '../../store/project-panel/project-panel-action';

export const MOVE_COLLECTION_DIALOG = 'moveCollectionDialog';

export const openMoveCollectionDialog = (resource: { name: string, uuid: string }) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(MOVE_COLLECTION_DIALOG, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: MOVE_COLLECTION_DIALOG, data: {} }));
    };

export const moveCollection = (resource: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(MOVE_COLLECTION_DIALOG));
        try {
            const collection = await services.collectionService.get(resource.uuid);
            await services.collectionService.update(resource.uuid, { ...collection, ownerUuid: resource.ownerUuid });
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(dialogActions.CLOSE_DIALOG({ id: MOVE_COLLECTION_DIALOG }));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection has been moved', hideDuration: 2000 }));
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(MOVE_COLLECTION_DIALOG, { ownerUuid: 'A collection with the same name already exists in the target project.' }));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: MOVE_COLLECTION_DIALOG }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not move the collection.', hideDuration: 2000 }));
            }
        }
    };

export const MoveCollectionDialog = compose(
    withDialog(MOVE_COLLECTION_DIALOG),
    reduxForm<MoveToFormDialogData>({
        form: MOVE_COLLECTION_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(moveCollection(data));
        }
    })
)(MoveToFormDialog);
