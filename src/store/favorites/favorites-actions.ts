// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { Dispatch } from "redux";
import { RootState } from "../store";
import { getUserUuid } from "common/getuser";
import { checkFavorite } from "./favorites-reducer";
import { snackbarActions, SnackbarKind } from "../snackbar/snackbar-actions";
import { ServiceRepository } from "services/services";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export const favoritesActions = unionize({
    TOGGLE_FAVORITE: ofType<{ resourceUuid: string }>(),
    CHECK_PRESENCE_IN_FAVORITES: ofType<string[]>(),
    UPDATE_FAVORITES: ofType<Record<string, boolean>>()
});

export type FavoritesAction = UnionOf<typeof favoritesActions>;

export const toggleFavorite = (resource: { uuid: string; name: string }) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        const userUuid = getUserUuid(getState());
        if (!userUuid) {
            return Promise.reject("No user");
        }
        dispatch(progressIndicatorActions.START_WORKING("toggleFavorite"));
        dispatch(favoritesActions.TOGGLE_FAVORITE({ resourceUuid: resource.uuid }));
        const isFavorite = checkFavorite(resource.uuid, getState().favorites);
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: isFavorite
                ? "Removing from favorites..."
                : "Adding to favorites...",
            kind: SnackbarKind.INFO
        }));

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
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING("toggleFavorite"));
            })
            .catch((e: any) => {
                dispatch(progressIndicatorActions.STOP_WORKING("toggleFavorite"));
                throw e;
            });
    };

export const updateFavorites = (resourceUuids: string[]) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (!userUuid) { return; }
        dispatch(favoritesActions.CHECK_PRESENCE_IN_FAVORITES(resourceUuids));
        services.favoriteService
            .checkPresenceInFavorites(userUuid, resourceUuids)
            .then((results: any) => {
                dispatch(favoritesActions.UPDATE_FAVORITES(results));
            });
    };
