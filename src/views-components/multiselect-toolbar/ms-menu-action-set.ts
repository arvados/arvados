// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { IconType } from "components/icon/icon";
import { ResourcesState } from "store/resources/resources";
import { FavoritesState } from "store/favorites/favorites-reducer";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";

export const MultiSelectMenuActionNames = {
  MAKE_A_COPY: "Make a copy",
  MOVE_TO: "Move to",
  ADD_TO_TRASH: "Add to Trash",
  ADD_TO_FAVORITES: "Add to Favorites",
  COPY_TO_CLIPBOARD: "Copy to clipboard",
  COPY_AND_RERUN_PROCESS: "Copy and re-run process",
  REMOVE: "Remove",
};

export type MultiSelectMenuAction = {
    name: string;
    icon: IconType;
    hasAlts: boolean;
    altName?: string;
    altIcon?: IconType;
    isForMulti: boolean;
    useAlts?: (uuid: string | null, resources: ResourcesState, favorites: FavoritesState) => boolean;
    execute(dispatch: Dispatch, resources: ContextMenuResource[], state?: any): void;
    adminOnly?: boolean;
};

export type MultiSelectMenuActionSet = MultiSelectMenuAction[][];
