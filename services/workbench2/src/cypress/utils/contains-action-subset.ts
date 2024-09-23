// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuAction } from "../../views-components/context-menu/context-menu-action-set";

export const containsActionSubSet = (mainSet: ContextMenuAction[], subSet: ContextMenuAction[]) => {
    const mainNames = mainSet.map(action => action.name)
    const subNames = subSet.map(action => action.name)
    return subNames.every(name => mainNames.includes(name));
}
