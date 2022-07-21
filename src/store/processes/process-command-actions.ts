// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from 'store/dialog/dialog-actions';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { getProcess } from 'store/processes/process';
import shellescape from 'shell-escape';

export const PROCESS_COMMAND_DIALOG_NAME = 'processCommandDialog';

export interface ProcessCommandDialogData {
    command: string;
    processName: string;
}

export const openProcessCommandDialog = (processUuid: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState) => {
        const process = getProcess(processUuid)(getState().resources);
        if (process) {
            const data: ProcessCommandDialogData = {
                command: shellescape(process.containerRequest.command),
                processName: process.containerRequest.name,
            };
            dispatch(dialogActions.OPEN_DIALOG({ id: PROCESS_COMMAND_DIALOG_NAME, data }));
        }
    };
