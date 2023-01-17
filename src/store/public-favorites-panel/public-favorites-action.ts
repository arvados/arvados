// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";

export const PUBLIC_FAVORITE_PANEL_ID = "publicFavoritePanel";
export const publicFavoritePanelActions = bindDataExplorerActions(PUBLIC_FAVORITE_PANEL_ID);

export const loadPublicFavoritePanel = () => (dispatch: Dispatch) => {
    dispatch(publicFavoritePanelActions.RESET_EXPLORER_SEARCH_VALUE());
    dispatch(publicFavoritePanelActions.REQUEST_ITEMS());
};