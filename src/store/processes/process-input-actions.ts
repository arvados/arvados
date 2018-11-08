// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from '~/store/dialog/dialog-actions';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { getProcess, Process } from '~/store/processes/process';

export const PROCESS_INPUT_DIALOG_NAME = 'processInputDialog';

export const openProcessInputDialog = (processUuid: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState) => {
        const process = getProcess(processUuid)(getState().resources);
        if (process) {
            const data: Process = process;
            dispatch(dialogActions.OPEN_DIALOG({ id: PROCESS_INPUT_DIALOG_NAME, data }));
        }
    }; 