// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProgressIndicatorAction, progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";

export type ProgressIndicatorState = { id: string, working: boolean }[];

const initialState: ProgressIndicatorState = [];

export const progressIndicatorReducer = (state: ProgressIndicatorState = initialState, action: ProgressIndicatorAction) => {
    const startWorking = (id: string) => state.find(p => p.working) ? state : state.concat({ id, working: true });
    const stopWorking = (id: string) => state.filter(p => p.id !== id);

    return progressIndicatorActions.match(action, {
        START: id => startWorking(id),
        STOP: id => stopWorking(id),
        PERSIST_STOP: id => state.map(p => ({
            id,
            working: p.id === id ? false : p.working
        })),
        TOGGLE: ({ id, working }) => working ? startWorking(id) : stopWorking(id),
        default: () => state,
    });
};

export function isSystemWorking(state: ProgressIndicatorState): boolean {
    return state.length > 0 && state.reduce((working, p) => working ? true : p.working, false);
}
