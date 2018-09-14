// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProgressIndicatorAction, progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';

export interface ProgressIndicatorState {
    'sidePanelProgress': { started: boolean };
    'contentProgress': { started: boolean };
    // 'workbenchProgress': { started: boolean };
}

const initialState: ProgressIndicatorState = {
    'sidePanelProgress': { started: false },
    'contentProgress': { started: false },
    // 'workbenchProgress': { started: false }
};

export enum ProgressIndicatorData {
    SIDE_PANEL_PROGRESS = 'sidePanelProgress',
    CONTENT_PROGRESS = 'contentProgress',
    // WORKBENCH_PROGRESS = 'workbenchProgress',
}

export const progressIndicatorReducer = (state: ProgressIndicatorState = initialState, action: ProgressIndicatorAction) => {
    return progressIndicatorActions.match(action, {
        START_SUBMIT: ({ id }) => ({ ...state, [id]: { started: true } }),
        STOP_SUBMIT: ({ id }) => ({
            ...state,
            [id]: state[id] ? { ...state[id], started: false } : { started: false }
        }),
        default: () => state,
    });
};

// export const getProgress = () =>
//     (dispatch: Dispatch, getState: () => RootState) => {
//         const progress = getState().progressIndicator;
//         if (progress.sidePanelProgress.started || progress.contentProgress.started) {
//             dispatch(progressIndicatorActions.START_SUBMIT({ id: ProgressIndicatorData.WORKBENCH_PROGRESS }));
//         } else {
//             dispatch(progressIndicatorActions.STOP_SUBMIT({ id: ProgressIndicatorData.WORKBENCH_PROGRESS }));
//         }
//     };
