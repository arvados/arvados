// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { IconType } from "components/icon/icon";
import { ResourcesState } from "store/resources/resources";
import { FavoritesState } from "store/favorites/favorites-reducer";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";

export const MultiSelectMenuActionNames = {
  ADD_TO_FAVORITES: "Add to Favorites",
  ADD_TO_TRASH: "Add to Trash",
  API_DETAILS: 'API Details',
  COPY_AND_RERUN_PROCESS: "Copy and re-run process",
  COPY_TO_CLIPBOARD: "Copy to clipboard",
  DELETE_WORKFLOW: 'DELETE_WORKFLOW', 
  EDIT_PPROJECT: 'Edit project',
  FREEZE_PROJECT: 'Freeze Project',
  MAKE_A_COPY: "Make a copy",
  MOVE_TO: "Move to",
  NEW_PROJECT: 'New project',
  OPEN_IN_NEW_TAB: 'Open in new tab',
  OPEN_W_3RD_PARTY_CLIENT: 'Open with 3rd party client',
  REMOVE: "Remove",
  RUN_WORKFLOW: 'RUN_WORKFLOW',
  SHARE: 'Share',
  VIEW_DETAILS: 'View details',
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
