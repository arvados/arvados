// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";

export const sidePanelActions = unionize({
    TOGGLE_SIDE_PANEL_ITEM_OPEN: ofType<string>(),
    TOGGLE_SIDE_PANEL_ITEM_ACTIVE: ofType<string>(),
    RESET_SIDE_PANEL_ACTIVITY: ofType<{}>(),
}, {
    tag: 'type',
    value: 'payload'
});

export type SidePanelAction = UnionOf<typeof sidePanelActions>;
