// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";

export const TRASH_PANEL_ID = "trashPanel";
export const trashPanelActions = bindDataExplorerActions(TRASH_PANEL_ID);
