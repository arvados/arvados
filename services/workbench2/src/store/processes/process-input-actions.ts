// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from 'store/dialog/dialog-actions';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { getProcess, Process } from 'store/processes/process';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getWorkflowInputs } from 'models/workflow';
import { JSONMount } from 'models/mount-types';
import { MOUNT_PATH_CWL_WORKFLOW } from 'models/process';

export const PROCESS_INPUT_DIALOG_NAME = 'processInputDialog';

export const openProcessInputDialog = (processUuid: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState) => {
        const process = getProcess(processUuid)(getState().resources);
        if (process) {
            const data: any = process;
            const inputs = getInputsFromWFMount(process);
            if (inputs && inputs.length > 0) {
                dispatch(dialogActions.OPEN_DIALOG({ id: PROCESS_INPUT_DIALOG_NAME, data }));
            } else {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'There are no inputs in this process!', kind: SnackbarKind.ERROR }));
            }
        }
    };

const getInputsFromWFMount = (process: Process) => {
    if (!process || !process.containerRequest.mounts[MOUNT_PATH_CWL_WORKFLOW] ) { return undefined; }
    const mnt = process.containerRequest.mounts[MOUNT_PATH_CWL_WORKFLOW] as JSONMount;
    return getWorkflowInputs(mnt.content);
};