// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";

export const PUBLIC_FAVORITE_PANEL_ID = "publicFavoritePanel";
export const publicFavoritePanelActions = bindDataExplorerActions(PUBLIC_FAVORITE_PANEL_ID);

export const loadPublicFavoritePanel = () => publicFavoritePanelActions.REQUEST_ITEMS();