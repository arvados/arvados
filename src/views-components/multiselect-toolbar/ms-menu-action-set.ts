// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { IconType } from "components/icon/icon";
import { ContextMenuAction } from "../context-menu/context-menu-action-set";
import { ResourcesState } from "store/resources/resources";

export const MultiSelectMenuActionNames = {
  MAKE_A_COPY: "Make a copy",
  MOVE_TO: "Move to",
  ADD_TO_TRASH: "Add to Trash",
  ADD_TO_FAVORITES: "Add to Favorites",
  COPY_TO_CLIPBOARD: "Copy to clipboard",
  COPY_AND_RERUN_PROCESS: "Copy and re-run process",
  REMOVE: "Remove",
};

export interface MultiSelectMenuAction extends ContextMenuAction {
    defaultText?: string;
    altText?: string;
    altIcon?: IconType;
    isDefault?: (uuid: string | null, resources: ResourcesState, favorites) => boolean;
    isForMulti: boolean;
}

export type MultiSelectMenuActionSet = MultiSelectMenuAction[][];
