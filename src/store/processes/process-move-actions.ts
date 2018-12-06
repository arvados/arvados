// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { startSubmit, stopSubmit, initialize, FormErrors } from 'redux-form';
import { ServiceRepository } from '~/services/services';
import { RootState } from '~/store/store';
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/services/common-service/common-resource-service";
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';
import { resetPickerProjectTree } from '~/store/project-tree-picker/project-tree-picker-actions';
import { projectPanelActions } from '~/store/project-panel/project-panel-action';
import { getProcess } from '~/store/processes/process';
import { initProjectsTreePicker } from '~/store/tree-picker/tree-picker-actions';

export const PROCESS_MOVE_FORM_NAME = 'processMoveFormName';

export const openMoveProcessDialog = (resource: { name: string, uuid: string }) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const process = getProcess(resource.uuid)(getState().resources);
        if (process) {
            dispatch<any>(resetPickerProjectTree());
            dispatch<any>(initProjectsTreePicker(PROCESS_MOVE_FORM_NAME));
            dispatch(initialize(PROCESS_MOVE_FORM_NAME, resource));
            dispatch(dialogActions.OPEN_DIALOG({ id: PROCESS_MOVE_FORM_NAME, data: {} }));
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process not found', hideDuration: 2000 }));
        }
    };

export const moveProcess = (resource: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(PROCESS_MOVE_FORM_NAME));
        try {
            const process = await services.containerRequestService.get(resource.uuid);
            await services.containerRequestService.update(resource.uuid, { ownerUuid: resource.ownerUuid });
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROCESS_MOVE_FORM_NAME }));
            return process;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(PROCESS_MOVE_FORM_NAME, { ownerUuid: 'A process with the same name already exists in the target project.' } as FormErrors));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: PROCESS_MOVE_FORM_NAME }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not move the process.', hideDuration: 2000 }));
            }
            return;
        }
    };
