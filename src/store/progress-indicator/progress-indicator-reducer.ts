// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProgressIndicatorAction, progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";

export type ProgressIndicatorState = { id: string, working: boolean }[];

const initialState: ProgressIndicatorState = [];

export const progressIndicatorReducer = (state: ProgressIndicatorState = initialState, action: ProgressIndicatorAction) => {
    const startWorking = (id: string) => state.find(p => p.id === id) ? state : state.concat({ id, working: true });
    const stopWorking = (id: string) => state.filter(p => p.id !== id);

    return progressIndicatorActions.match(action, {
        START_WORKING: id => startWorking(id),
        STOP_WORKING: id => stopWorking(id),
        PERSIST_STOP_WORKING: id => state.map(p => ({
            ...p,
            working: p.id === id ? false : p.working
        })),
        TOGGLE_WORKING: ({ id, working }) => working ? startWorking(id) : stopWorking(id),
        default: () => state,
    });
};

export function isSystemWorking(state: ProgressIndicatorState): boolean {
    return state.length > 0 && state.reduce((working, p) => working ? true : p.working, false);
}
