// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PublicFavoritesAction, publicFavoritesActions } from "./public-favorites-actions";

export type PublicFavoritesState = Record<string, boolean>;

export const publicFavoritesReducer = (state: PublicFavoritesState = {}, action: PublicFavoritesAction) =>
    publicFavoritesActions.match(action, {
        UPDATE_PUBLIC_FAVORITES: publicFavorites => ({ ...state, ...publicFavorites }),
        default: () => state
    });

export const checkPublicFavorite = (uuid: string, state: PublicFavoritesState) => state[uuid] === true;