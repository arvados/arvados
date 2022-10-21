// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { ContextMenuItem } from "components/context-menu/context-menu";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { RootState } from "store/store";

export interface ContextMenuAction extends ContextMenuItem {
    execute(dispatch: Dispatch, resource: ContextMenuResource, state?: any): void;
}

export type ContextMenuActionSet = Array<Array<ContextMenuAction>>;
