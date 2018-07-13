// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuItemGroup } from "../../../components/context-menu/context-menu";
import { ContextMenuItemSet } from "../context-menu-item-set";

export const emptyItemSet: ContextMenuItemSet = {
    getItems: () => items,
    handleItem: () => { return; }
};

const items: ContextMenuItemGroup[] = [];