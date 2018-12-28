// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from '~/store/dialog/dialog-actions';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { getProcess } from '~/store/processes/process';
import {snackbarActions, SnackbarKind} from '~/store/snackbar/snackbar-actions';

export const PROCESS_INPUT_DIALOG_NAME = 'processInputDialog';

export const openProcessInputDialog = (processUuid: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState) => {
        const process = getProcess(processUuid)(getState().resources);
        if (process) {
            const data: any = process;
            if (data && data.containerRequest.mounts.varLibCwlWorkflowJson && data.containerRequest.mounts.varLibCwlWorkflowJson.content.graph[1].inputs.length > 0) {
                dispatch(dialogActions.OPEN_DIALOG({ id: PROCESS_INPUT_DIALOG_NAME, data }));
            } else {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'There are no inputs in this process!', kind: SnackbarKind.ERROR }));
            }
        }
    }; 