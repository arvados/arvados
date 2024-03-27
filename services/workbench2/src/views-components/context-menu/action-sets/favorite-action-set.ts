// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from '../context-menu-action-set';
import { ToggleFavoriteAction } from '../actions/favorite-action';
import { toggleFavorite } from 'store/favorites/favorites-actions';
import { favoritePanelActions } from 'store/favorite-panel/favorite-panel-action';

export const favoriteActionSet: ContextMenuActionSet = [
    [
        {
            component: ToggleFavoriteAction,
            name: 'Add to Favorites',
            execute: (dispatch, resources) => {
                resources.forEach((resource) =>
                    dispatch<any>(toggleFavorite(resource)).then(() => {
                        dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
                    })
                );
            },
        },
    ],
];
