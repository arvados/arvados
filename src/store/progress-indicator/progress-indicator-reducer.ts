// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProgressIndicatorAction, progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";

export interface ProgressIndicatorState {
    'workbenchProgress': { started: boolean };
    'contentProgress': { started: boolean };
    'detailsProgress': { started: boolean };
}

const initialState: ProgressIndicatorState = {
    'workbenchProgress': { started: false },
    'contentProgress': { started: false },
    'detailsProgress': { started: false }
};

export enum ProgressIndicatorData {
    WORKBENCH_PROGRESS = 'workbenchProgress',
    CONTENT_PROGRESS = 'contentProgress',
    DETAILS_PROGRESS = 'detailsProgress'
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
