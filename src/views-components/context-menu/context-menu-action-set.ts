// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { ContextMenuItem } from "components/context-menu/context-menu";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";

export interface ContextMenuActionItem extends ContextMenuItem {
    execute(dispatch: Dispatch, resources: ContextMenuResource[], state?: any): void;
}

export type ContextMenuActionItemSet = Array<Array<ContextMenuActionItem>>;
