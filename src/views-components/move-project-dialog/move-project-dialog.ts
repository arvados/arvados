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

export const MOVE_PROJECT_DIALOG = 'moveProjectDialog';

export const openMoveProjectDialog = (resource: { name: string, uuid: string }) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(MOVE_PROJECT_DIALOG, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: MOVE_PROJECT_DIALOG, data: {} }));
    };

export const moveProject = (resource: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(MOVE_PROJECT_DIALOG));
        try {
            const project = await services.projectService.get(resource.uuid);
            await services.projectService.update(resource.uuid, { ...project, ownerUuid: resource.ownerUuid });
            dispatch(dialogActions.CLOSE_DIALOG({ id: MOVE_PROJECT_DIALOG }));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Project has been moved', hideDuration: 2000 }));
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(MOVE_PROJECT_DIALOG, { ownerUuid: 'A project with the same name already exists in the target project.' }));
            } else if (error === CommonResourceServiceError.OWNERSHIP_CYCLE) {
                dispatch(stopSubmit(MOVE_PROJECT_DIALOG, { ownerUuid: 'Cannot move a project into itself.' }));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: MOVE_PROJECT_DIALOG }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not move the project.', hideDuration: 2000 }));
            }
        }
    };

export const MoveProjectDialog = compose(
    withDialog(MOVE_PROJECT_DIALOG),
    reduxForm<MoveToFormDialogData>({
        form: MOVE_PROJECT_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(moveProject(data));
        }
    })
)(MoveToFormDialog);

