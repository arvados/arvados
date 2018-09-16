// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProgressIndicatorAction, progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";

export interface ProgressIndicatorState {
    [key: string]: {
        working: boolean
    };
}

const initialState: ProgressIndicatorState = {
};

export const progressIndicatorReducer = (state: ProgressIndicatorState = initialState, action: ProgressIndicatorAction) => {
    return progressIndicatorActions.match(action, {
        START: id => ({ ...state, [id]: { working: true } }),
        STOP: id => ({ ...state, [id]: { working: false } }),
        TOGGLE: ({ id, working }) => ({ ...state, [id]: { working }}),
        default: () => state,
    });
};

export function isSystemWorking(state: ProgressIndicatorState): boolean {
    return Object.keys(state).reduce((working, k) => working ? true : state[k].working, false);
}
