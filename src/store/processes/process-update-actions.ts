// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { initialize, startSubmit, stopSubmit } from 'redux-form';
import { RootState } from "~/store/store";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/services/common-service/common-resource-service";
import { ServiceRepository } from "~/services/services";
import { getProcess } from '~/store/processes/process';
import { projectPanelActions } from '~/store/project-panel/project-panel-action';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';

export interface ProcessUpdateFormDialogData {
    uuid: string;
    name: string;
}

export const PROCESS_UPDATE_FORM_NAME = 'processUpdateFormName';

export const openProcessUpdateDialog = (resource: ProcessUpdateFormDialogData) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const process = getProcess(resource.uuid)(getState().resources);
        if(process) {
            dispatch(initialize(PROCESS_UPDATE_FORM_NAME, { ...resource, name: process.containerRequest.name }));
            dispatch(dialogActions.OPEN_DIALOG({ id: PROCESS_UPDATE_FORM_NAME, data: {} }));
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process not found', hideDuration: 2000 }));
        }
    };

export const updateProcess = (resource: ProcessUpdateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(PROCESS_UPDATE_FORM_NAME));
        try {
            const process = await services.containerRequestService.get(resource.uuid);
            const updatedProcess = await services.containerRequestService.update(resource.uuid, { ...process, name: resource.name });
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROCESS_UPDATE_FORM_NAME }));
            return updatedProcess;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(PROCESS_UPDATE_FORM_NAME, { name: 'Process with the same name already exists.' }));
            } else if (error === CommonResourceServiceError.MODIFYING_CONTAINER_REQUEST_FINAL_STATE) {
                dispatch(stopSubmit(PROCESS_UPDATE_FORM_NAME, { name: 'You cannot modified in "Final" state.' }));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: PROCESS_UPDATE_FORM_NAME }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not update the process.', hideDuration: 2000 }));
            }
            return;
        }
    };