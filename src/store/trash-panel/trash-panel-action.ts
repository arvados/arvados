// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";

export const TRASH_PANEL_ID = "trashPanel";
export const trashPanelActions = bindDataExplorerActions(TRASH_PANEL_ID);

export const loadTrashPanel = () => trashPanelActions.REQUEST_ITEMS();
