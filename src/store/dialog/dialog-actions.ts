// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";

export const dialogActions = unionize({
    OPEN_DIALOG: ofType<{ id: string, data: any }>(),
    CLOSE_DIALOG: ofType<{ id: string }>()
}, {
        tag: 'type',
        value: 'payload'
    });

export type DialogAction = UnionOf<typeof dialogActions>;
