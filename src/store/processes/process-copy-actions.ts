// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { dialogActions } from 'store/dialog/dialog-actions';
import { initialize, startSubmit } from 'redux-form';
import { resetPickerProjectTree } from 'store/project-tree-picker/project-tree-picker-actions';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { CopyFormDialogData } from 'store/copy-dialog/copy-dialog';
import { Process, getProcess } from 'store/processes/process';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { initProjectsTreePicker } from 'store/tree-picker/tree-picker-actions';
import { ContainerRequestResource, ContainerRequestState } from 'models/container-request';

export const PROCESS_COPY_FORM_NAME = 'processCopyFormName';
export const MULTI_PROCESS_COPY_FORM_NAME = 'multiProcessCopyFormName';

export const openCopyProcessDialog = (resource: { name: string; uuid: string }) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const process = getProcess(resource.uuid)(getState().resources);
    if (process) {
        dispatch<any>(resetPickerProjectTree());
        dispatch<any>(initProjectsTreePicker(PROCESS_COPY_FORM_NAME));
        const initialData: CopyFormDialogData = { name: `Copy of: ${resource.name}`, uuid: resource.uuid, ownerUuid: '' };
        dispatch<any>(initialize(PROCESS_COPY_FORM_NAME, initialData));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROCESS_COPY_FORM_NAME, data: {} }));
    } else {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process not found', hideDuration: 2000, kind: SnackbarKind.ERROR }));
    }
};

export const openCopyManyProcessesDialog = (list: Array<string>) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    //need to test with undefined process array
    const testList = ['tordo-xvhdp-f8ps1vs3s3c2u3v', 'foo'];
    const processes = list.map((uuid) => {
        // const processes = testList.map((uuid) => {
        const process = getProcess(uuid)(getState().resources);
        if (!process) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: `Process ${uuid} not found`, hideDuration: 2000, kind: SnackbarKind.ERROR }));
            return undefined;
        } else return process;
    });
    console.log(processes);

    let initialData: CopyFormDialogData;
    if (processes.every((process) => !!process)) {
        const { name, uuid } = (processes[0] as Process).containerRequest;
        dispatch<any>(resetPickerProjectTree());
        dispatch<any>(initProjectsTreePicker(MULTI_PROCESS_COPY_FORM_NAME));
        initialData = { name: `Copy of: ${name}`, uuid: uuid, ownerUuid: '' };
        dispatch<any>(initialize(MULTI_PROCESS_COPY_FORM_NAME, initialData));
        dispatch(dialogActions.OPEN_DIALOG({ id: MULTI_PROCESS_COPY_FORM_NAME, data: {} }));
    }
};

export const copyProcess = (resource: CopyFormDialogData) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch(startSubmit(PROCESS_COPY_FORM_NAME));
    try {
        const process = await services.containerRequestService.get(resource.uuid);
        const {
            command,
            containerCountMax,
            containerImage,
            cwd,
            description,
            environment,
            kind,
            mounts,
            outputName,
            outputPath,
            outputProperties,
            outputStorageClasses,
            outputTtl,
            properties,
            runtimeConstraints,
            schedulingParameters,
            useExisting,
        } = process;
        const newProcess = await services.containerRequestService.create({
            command,
            containerCountMax,
            containerImage,
            cwd,
            description,
            environment,
            kind,
            mounts,
            name: resource.name,
            outputName,
            outputPath,
            outputProperties,
            outputStorageClasses,
            outputTtl,
            ownerUuid: resource.ownerUuid,
            priority: 500,
            properties,
            runtimeConstraints,
            schedulingParameters,
            state: ContainerRequestState.UNCOMMITTED,
            useExisting,
        });
        dispatch(dialogActions.CLOSE_DIALOG({ id: PROCESS_COPY_FORM_NAME }));
        return newProcess;
    } catch (e) {
        dispatch(dialogActions.CLOSE_DIALOG({ id: PROCESS_COPY_FORM_NAME }));
        throw new Error('Could not copy the process.');
    }
};
