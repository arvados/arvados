// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "./favorite-action";
import { toggleFavorite } from "../../../store/favorites/favorites-actions";
import { dataExplorerActions } from "../../../store/data-explorer/data-explorer-action";
import { FAVORITE_PANEL_ID } from "../../../views/favorite-panel/favorite-panel";

export const favoriteActionSet: ContextMenuActionSet = [[{
    component: ToggleFavoriteAction,
    execute: (dispatch, resource) => {
        debugger;
        dispatch<any>(toggleFavorite(resource)).then(() => {
            dispatch<any>(dataExplorerActions.REQUEST_ITEMS({ id : FAVORITE_PANEL_ID }));
        });
    }
}]];
