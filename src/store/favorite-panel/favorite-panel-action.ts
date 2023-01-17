// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";

export const FAVORITE_PANEL_ID = "favoritePanel";
export const favoritePanelActions = bindDataExplorerActions(FAVORITE_PANEL_ID);

export const loadFavoritePanel = () => (dispatch: Dispatch) => {
    dispatch(favoritePanelActions.RESET_EXPLORER_SEARCH_VALUE());
    dispatch(favoritePanelActions.REQUEST_ITEMS());
};