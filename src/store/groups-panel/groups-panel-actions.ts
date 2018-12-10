// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "~/store/data-explorer/data-explorer-action";

export const GROUPS_PANEL_ID = "groupsPanel";
export const GroupsPanelActions = bindDataExplorerActions(GROUPS_PANEL_ID);

export const loadGroupsPanel = () => GroupsPanelActions.REQUEST_ITEMS();
