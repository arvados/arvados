// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { TogglePublicFavoriteAction } from "views-components/context-menu/actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";

import { projectActionSet, filterGroupActionSet } from "views-components/context-menu/action-sets/project-action-set";

export const projectAdminActionSet: ContextMenuActionSet = [[
    ...projectActionSet.reduce((prev, next) => prev.concat(next), []),
    {
        component: TogglePublicFavoriteAction,
        name: 'TogglePublicFavoriteAction',
        execute: (dispatch, resource) => {
            dispatch<any>(togglePublicFavorite(resource)).then(() => {
                dispatch<any>(publicFavoritePanelActions.REQUEST_ITEMS());
            });
        }
    }
]];

export const filterGroupAdminActionSet: ContextMenuActionSet = [[
    ...filterGroupActionSet.reduce((prev, next) => prev.concat(next), []),
    {
        component: TogglePublicFavoriteAction,
        name: 'TogglePublicFavoriteAction',
        execute: (dispatch, resource) => {
            dispatch<any>(togglePublicFavorite(resource)).then(() => {
                dispatch<any>(publicFavoritePanelActions.REQUEST_ITEMS());
            });
        }
    }
]];
