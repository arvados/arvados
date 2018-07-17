// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { ContextMenuPosition, ContextMenuResource } from "./context-menu-reducer";

const actions = unionize({
    OPEN_CONTEXT_MENU: ofType<{ position: ContextMenuPosition, resource: ContextMenuResource }>(),
    CLOSE_CONTEXT_MENU: ofType<{}>()
}, {
        tag: 'type',
        value: 'payload'
    });

export type ContextMenuAction = UnionOf<typeof actions>;
export default actions;