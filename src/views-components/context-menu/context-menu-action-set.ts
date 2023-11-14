// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { ContextMenuItem } from "components/context-menu/context-menu";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { IconType } from "components/icon/icon";

export interface ContextMenuAction extends ContextMenuItem {
    execute(dispatch: Dispatch, resources: ContextMenuResource[], state?: any): void;
}

export interface MultiSelectMenuAction extends ContextMenuAction {
    defaultText?: string
    defaultIcon?: IconType
    altText?: string
    altIcon?: IconType
}

export type ContextMenuActionSet = Array<Array<ContextMenuAction>>;
export type MultiSelectMenuActionSet = Array<Array<MultiSelectMenuAction>>;
