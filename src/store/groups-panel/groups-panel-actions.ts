// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "~/store/data-explorer/data-explorer-action";
import { dialogActions } from '~/store/dialog/dialog-actions';

export const GROUPS_PANEL_ID = "groupsPanel";
export const CREATE_GROUP_DIALOG = "createGroupDialog";
export const CREATE_GROUP_FORM = "createGroupForm";

export const GroupsPanelActions = bindDataExplorerActions(GROUPS_PANEL_ID);

export const loadGroupsPanel = () => GroupsPanelActions.REQUEST_ITEMS();

export const openCreateGroupDialog = () =>
    dialogActions.OPEN_DIALOG({ id: CREATE_GROUP_DIALOG, data: {} });
