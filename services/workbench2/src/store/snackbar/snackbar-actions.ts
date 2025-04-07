// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";

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
    CLOSE_SNACKBAR: ofType<{}|null>(),
    SHIFT_MESSAGES: ofType<{}>()
});

export const showSuccessSnackbar = (message: string) =>
    snackbarActions.OPEN_SNACKBAR({ message, hideDuration: 2000, kind: SnackbarKind.SUCCESS });

export const showErrorSnackbar = (message: string) =>
    snackbarActions.OPEN_SNACKBAR({ message, hideDuration: 4000, kind: SnackbarKind.ERROR });

export type SnackbarAction = UnionOf<typeof snackbarActions>;
