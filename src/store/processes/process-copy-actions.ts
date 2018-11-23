// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { initialize, startSubmit } from 'redux-form';
import { resetPickerProjectTree } from '~/store/project-tree-picker/project-tree-picker-actions';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { CopyFormDialogData } from '~/store/copy-dialog/copy-dialog';
import { getProcess, ProcessStatus, getProcessStatus } from '~/store/processes/process';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { initProjectsTreePicker } from '~/store/tree-picker/tree-picker-actions';

export const PROCESS_COPY_FORM_NAME = 'processCopyFormName';

export const openCopyProcessDialog = (resource: { name: string, uuid: string }) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const process = getProcess(resource.uuid)(getState().resources);
        if (process) {
            const processStatus = getProcessStatus(process);
            if (processStatus) {
                dispatch<any>(resetPickerProjectTree());
                dispatch<any>(initProjectsTreePicker(PROCESS_COPY_FORM_NAME));
                const initialData: CopyFormDialogData = { name: `Copy of: ${resource.name}`, uuid: resource.uuid, ownerUuid: '' };
                dispatch<any>(initialize(PROCESS_COPY_FORM_NAME, initialData));
                dispatch(dialogActions.OPEN_DIALOG({ id: PROCESS_COPY_FORM_NAME, data: {} }));
            } else {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'You can copy only draft processes.', hideDuration: 2000 }));
            }
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process not found', hideDuration: 2000 }));
        }
    };

export const copyProcess = (resource: CopyFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(PROCESS_COPY_FORM_NAME));
        try {
            const process = await services.containerRequestService.get(resource.uuid);
            const uuidKey = '';
            process.uuid = uuidKey;
            await services.containerRequestService.create({ command: process.command, containerImage: process.containerImage, outputPath: process.outputPath, ownerUuid: resource.ownerUuid, name: resource.name });
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROCESS_COPY_FORM_NAME }));
            return process;
        } catch (e) {
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROCESS_COPY_FORM_NAME }));
            throw new Error('Could not copy the process.');
        }
    };