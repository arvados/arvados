// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import { RenameIcon, AdvancedIcon, RemoveIcon, DetailsIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openRemoveGroupDialog, openGroupUpdateDialog } from 'store/groups-panel/groups-panel-actions';
import { MultiSelectMenuAction, MultiSelectMenuActionSet } from 'views-components/multiselect-toolbar/ms-menu-actions';
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';

const msRenameGroupAction: MultiSelectMenuAction = {
    name: ContextMenuActionNames.RENAME,
    icon: RenameIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openGroupUpdateDialog(resources[0]))
    },
};

const msAdvancedAction: MultiSelectMenuAction  = {
  name: ContextMenuActionNames.API_DETAILS,
  icon: AdvancedIcon,
  hasAlts: false,
  isForMulti: false,
  execute: (dispatch, resources) => {
      dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
  },
};

const msViewDetailsAction: MultiSelectMenuAction  = {
  name: ContextMenuActionNames.VIEW_DETAILS,
  icon: DetailsIcon,
  hasAlts: false,
  isForMulti: false,
  execute: (dispatch, resources) => {
      dispatch<any>(toggleDetailsPanel(resources[0].uuid));
  },
};

const msRemoveGroupAction: MultiSelectMenuAction = {
    name: ContextMenuActionNames.REMOVE,
    icon: RemoveIcon,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resources) => {
        resources.forEach(resource => dispatch<any>(openRemoveGroupDialog(resource.uuid, resources.length)));
    },
};

export const msGroupActionSet: MultiSelectMenuActionSet = [
  [
    msRenameGroupAction,
    msAdvancedAction,
    msRemoveGroupAction,
    msViewDetailsAction,
  ]
]