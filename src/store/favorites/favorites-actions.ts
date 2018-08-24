// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { RootState } from "../store";
import { checkFavorite } from "./favorites-reducer";
import { snackbarActions } from "../snackbar/snackbar-actions";
import { ServiceRepository } from "~/services/services";

export const favoritesActions = unionize({
    TOGGLE_FAVORITE: ofType<{ resourceUuid: string }>(),
    CHECK_PRESENCE_IN_FAVORITES: ofType<string[]>(),
    UPDATE_FAVORITES: ofType<Record<string, boolean>>()
}, { tag: 'type', value: 'payload' });

export type FavoritesAction = UnionOf<typeof favoritesActions>;

export const toggleFavorite = (resource: { uuid: string; name: string }) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        const userUuid = getState().auth.user!.uuid;
        dispatch(favoritesActions.TOGGLE_FAVORITE({ resourceUuid: resource.uuid }));
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Working..." }));
        const isFavorite = checkFavorite(resource.uuid, getState().favorites);
        const promise: any = isFavorite
            ? services.favoriteService.delete({ userUuid, resourceUuid: resource.uuid })
            : services.favoriteService.create({ userUuid, resource });

        return promise
            .then(() => {
                dispatch(favoritesActions.UPDATE_FAVORITES({ [resource.uuid]: !isFavorite }));
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: isFavorite
                        ? "Removed from favorites"
                        : "Added to favorites",
                    hideDuration: 2000
                }));
            });
    };

export const updateFavorites = (resourceUuids: string[]) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getState().auth.user!.uuid;
        dispatch(favoritesActions.CHECK_PRESENCE_IN_FAVORITES(resourceUuids));
        services.favoriteService
            .checkPresenceInFavorites(userUuid, resourceUuids)
            .then((results: any) => {
                dispatch(favoritesActions.UPDATE_FAVORITES(results));
            });
    };
