// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { FavoritesAction, favoritesActions } from "./favorites-actions";

export type FavoritesState = Record<string, boolean>;

export const favoritesReducer = (state: FavoritesState = {}, action: FavoritesAction) => 
    favoritesActions.match(action, {
        UPDATE_FAVORITES: favorites => ({...state, ...favorites}),
        default: () => state
    });

export const checkFavorite = (uuid: string, state: FavoritesState) => state[uuid] === true;