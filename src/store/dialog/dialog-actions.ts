// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";

export const dialogActions = unionize({
    OPEN_DIALOG: ofType<{ id: string, data: any }>(),
    CLOSE_DIALOG: ofType<{ id: string }>(),
    CLOSE_ALL_DIALOGS: ofType<{}>()
});

export type DialogAction = UnionOf<typeof dialogActions>;
