// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { ContextMenuKind } from 'views-components/context-menu/menu-item-sort';
import { sortMenuItems, menuDirection } from 'views-components/context-menu/menu-item-sort';

const menuActionSets = new Map<string, ContextMenuActionSet>();

export const addMenuActionSet = (name: ContextMenuKind, itemSet: ContextMenuActionSet) => {
    const sorted = itemSet.map(items => sortMenuItems(name, items, menuDirection.VERTICAL));
    menuActionSets.set(name, sorted);
};

const emptyActionSet: ContextMenuActionSet = [];
export const getMenuActionSet = (resource?: ContextMenuResource): ContextMenuActionSet =>
    resource ? menuActionSets.get(resource.menuKind) || emptyActionSet : emptyActionSet;

export const getMenuActionSetByKind = (kind: ContextMenuKind): ContextMenuActionSet =>
    menuActionSets.get(kind) || emptyActionSet;