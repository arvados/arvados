// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";

export enum SnackbarKind {
    SUCCESS,
    ERROR,
    INFO,
    WARNING
}

export const snackbarActions = unionize({
    OPEN_SNACKBAR: ofType<{message: string; hideDuration?: number, kind?: SnackbarKind}>(),
    CLOSE_SNACKBAR: ofType<{}>()
});

export type SnackbarAction = UnionOf<typeof snackbarActions>;
