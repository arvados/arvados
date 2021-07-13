// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";

export const resourceActionSet: ContextMenuActionSet = [[{
    component: ToggleFavoriteAction,
    execute: (dispatch, resource) => {
        dispatch<any>(toggleFavorite(resource));
    }
}]];
