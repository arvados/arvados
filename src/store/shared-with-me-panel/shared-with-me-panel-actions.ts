// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";

export const SHARED_WITH_ME_PANEL_ID = "sharedWithMePanel";
export const sharedWithMePanelActions = bindDataExplorerActions(SHARED_WITH_ME_PANEL_ID);
export const loadSharedWithMePanel = () => sharedWithMePanelActions.REQUEST_ITEMS();
