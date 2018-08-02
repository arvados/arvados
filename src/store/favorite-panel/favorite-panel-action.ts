// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";

export const FAVORITE_PANEL_ID = "favoritePanel";
export const favoritePanelActions = bindDataExplorerActions(FAVORITE_PANEL_ID);
