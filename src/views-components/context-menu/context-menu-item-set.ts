// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { ContextMenuItemGroup, ContextMenuItem } from "../../components/context-menu/context-menu";
import { ContextMenuResource } from "../../store/context-menu/context-menu-reducer";

export interface ContextMenuItemSet {
    handleItem (dispatch: Dispatch, action: ContextMenuItem, resource: ContextMenuResource): void;
    getItems (): ContextMenuItemGroup[];
}
