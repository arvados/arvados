// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";

export const snackbarActions = unionize({
    OPEN_SNACKBAR: ofType<{message: string; hideDuration?: number}>(),
    CLOSE_SNACKBAR: ofType<{}>()
}, { tag: 'type', value: 'payload' });

export type SnackbarAction = UnionOf<typeof snackbarActions>;
