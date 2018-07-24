// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { favoriteService } from "../../services/services";
import { RootState } from "../store";
import { checkFavorite } from "./favorites-reducer";

export const favoritesActions = unionize({
    TOGGLE_FAVORITE: ofType<{ resourceUuid: string }>(),
    CHECK_PRESENCE_IN_FAVORITES: ofType<string[]>(),
    UPDATE_FAVORITES: ofType<Record<string, boolean>>()
}, { tag: 'type', value: 'payload' });

export type FavoritesAction = UnionOf<typeof favoritesActions>;

export const toggleFavorite = (resource: { uuid: string; name: string }) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const userUuid = getState().auth.user!.uuid;
        dispatch(favoritesActions.TOGGLE_FAVORITE({ resourceUuid: resource.uuid }));
        const isFavorite = checkFavorite(resource.uuid, getState().favorites);
        const promise: (any) = isFavorite
            ? favoriteService.delete({ userUuid, resourceUuid: resource.uuid })
            : favoriteService.create({ userUuid, resource });

        promise
            .then(() => {
                dispatch(favoritesActions.UPDATE_FAVORITES({ [resource.uuid]: !isFavorite }));
            });
    };

export const checkPresenceInFavorites = (resourceUuids: string[]) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const userUuid = getState().auth.user!.uuid;
        dispatch(favoritesActions.CHECK_PRESENCE_IN_FAVORITES(resourceUuids));
        favoriteService
            .checkPresenceInFavorites(userUuid, resourceUuids)
            .then(results => {
                dispatch(favoritesActions.UPDATE_FAVORITES(results));
            });
    };

