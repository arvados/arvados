// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProgressIndicatorAction, progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export type ProgressIndicatorState = { id: string, working: boolean }[];

const initialState: ProgressIndicatorState = [];

export const progressIndicatorReducer = (state: ProgressIndicatorState = initialState, action: ProgressIndicatorAction) => {
    const stopWorking = (id: string) => state.filter(p => p.id !== id);

    return progressIndicatorActions.match(action, {
        START_WORKING: id => startWorking(id, state),
        STOP_WORKING: id => stopWorking(id),
        PERSIST_STOP_WORKING: id => state.map(p => ({
            ...p,
            working: p.id === id ? false : p.working
        })),
        TOGGLE_WORKING: ({ id, working }) => working ? startWorking(id, state) : stopWorking(id),
        default: () => state,
    });
};

const startWorking = (id: string, state: ProgressIndicatorState) => {
    return getProgressIndicator(id)(state)
        ? state.map(indicator => indicator.id === id
            ? { ...indicator, working: true }
            : indicator)
        : state.concat({ id, working: true });
};

export function isSystemWorking(state: ProgressIndicatorState): boolean {
    return state.length > 0 && state.reduce((working, p) => working ? true : p.working, false);
}

export const getProgressIndicator = (id: string) =>
    (state: ProgressIndicatorState) =>
        state.find(state => state.id === id);
