// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { SnackbarAction, snackbarActions } from "./snackbar-actions";

export interface SnackbarState {
    message: string;
    open: boolean;
    hideDuration: number;
}

const DEFAULT_HIDE_DURATION = 3000;

const initialState: SnackbarState = {
    message: "",
    open: false,
    hideDuration: DEFAULT_HIDE_DURATION
};

export const snackbarReducer = (state = initialState, action: SnackbarAction) => {
    return snackbarActions.match(action, {
        OPEN_SNACKBAR: data => ({ ...initialState, ...data, open: true }),
        CLOSE_SNACKBAR: () => initialState,
        default: () => state,
    });
};
