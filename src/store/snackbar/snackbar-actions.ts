// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";

export interface SnackbarMessage {
    message: string;
    hideDuration: number;
    kind: SnackbarKind;
    link?: string;
}

export enum SnackbarKind {
    SUCCESS = 1,
    ERROR = 2,
    INFO = 3,
    WARNING = 4
}

export const snackbarActions = unionize({
    OPEN_SNACKBAR: ofType<{message: string; hideDuration?: number, kind?: SnackbarKind, link?: string}>(),
    CLOSE_SNACKBAR: ofType<{}>(),
    SHIFT_MESSAGES: ofType<{}>()
});

export type SnackbarAction = UnionOf<typeof snackbarActions>;
