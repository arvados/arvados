// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PublicFavoritesState } from "./public-favorites";
import { PublicFavoritesAction, publicFavoritesActions } from "./public-favorites-actions";

export const publicFavoritesReducer = (state: PublicFavoritesState = {}, action: PublicFavoritesAction) =>
    publicFavoritesActions.match(action, {
        UPDATE_PUBLIC_FAVORITES: publicFavorites => ({ ...state, ...publicFavorites }),
        default: () => state
    });
