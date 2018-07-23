// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "../../../node_modules/redux";
import { favoriteService } from "../../services/services";

export const favoritesActions = unionize({
    CHECK_PRESENCE_IN_FAVORITES: ofType<string[]>(),
    UPDATE_FAVORITES: ofType<Record<string, boolean>>()
}, { tag: 'type', value: 'payload' });

export type FavoritesAction = UnionOf<typeof favoritesActions>;

export const checkPresenceInFavorites = (userUuid: string, resourceUuids: string[]) =>
    (dispatch: Dispatch) => {
        dispatch(favoritesActions.CHECK_PRESENCE_IN_FAVORITES(resourceUuids));
        favoriteService
            .checkPresenceInFavorites(userUuid, resourceUuids)
            .then(results => {
                dispatch(favoritesActions.UPDATE_FAVORITES(results));
            });
    };

