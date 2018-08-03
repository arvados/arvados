// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DialogAction, dialogActions } from "./dialog-actions";

export type DialogState = Record<string, Dialog>;

export interface Dialog {
    open: boolean;
    data?: any;
}

export const dialogReducer = (state: DialogState = {}, action: DialogAction) =>
    dialogActions.match(action, {
        OPEN_DIALOG: ({ id, data }) => ({ ...state, [id]: { open: true, data } }),
        CLOSE_DIALOG: ({ id }) => ({ 
            ...state, 
            [id]: state[id] ? { ...state[id], open: false } : { open: false } }),
        default: () => state,
    });

