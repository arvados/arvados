// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { Dispatch } from "redux";
import { RootState } from "../store";
import { checkPublicFavorite } from "./public-favorites-reducer";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { ServiceRepository } from "services/services";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export const publicFavoritesActions = unionize({
    TOGGLE_PUBLIC_FAVORITE: ofType<{ resourceUuid: string }>(),
    CHECK_PRESENCE_IN_PUBLIC_FAVORITES: ofType<string[]>(),
    UPDATE_PUBLIC_FAVORITES: ofType<Record<string, boolean>>()
});

export type PublicFavoritesAction = UnionOf<typeof publicFavoritesActions>;

export const togglePublicFavorite = (resource: { uuid: string; name: string }) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        dispatch(progressIndicatorActions.START_WORKING("togglePublicFavorite"));
        const uuidPrefix = getState().auth.config.uuidPrefix;
        const uuid = `${uuidPrefix}-j7d0g-publicfavorites`;
        dispatch(publicFavoritesActions.TOGGLE_PUBLIC_FAVORITE({ resourceUuid: resource.uuid }));
        const isPublicFavorite = checkPublicFavorite(resource.uuid, getState().publicFavorites);
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: isPublicFavorite
                ? "Removing from public favorites..."
                : "Adding to public favorites...",
            kind: SnackbarKind.INFO
        }));

        const promise: any = isPublicFavorite
            ? services.favoriteService.delete({ userUuid: uuid, resourceUuid: resource.uuid })
            : services.favoriteService.create({ userUuid: uuid, resource });

        return promise
            .then(() => {
                dispatch(publicFavoritesActions.UPDATE_PUBLIC_FAVORITES({ [resource.uuid]: !isPublicFavorite }));
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: isPublicFavorite
                        ? "Removed from public favorites"
                        : "Added to public favorites",
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch(progressIndicatorActions.STOP_WORKING("togglePublicFavorite"));
            })
            .catch((e: any) => {
                dispatch(progressIndicatorActions.STOP_WORKING("togglePublicFavorite"));
                throw e;
            });
    };

export const updatePublicFavorites = (resourceUuids: string[]) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuidPrefix = getState().auth.config.uuidPrefix;
        const uuid = `${uuidPrefix}-j7d0g-publicfavorites`;
        dispatch(publicFavoritesActions.CHECK_PRESENCE_IN_PUBLIC_FAVORITES(resourceUuids));
        services.favoriteService
            .checkPresenceInFavorites(uuid, resourceUuids)
            .then((results: any) => {
                dispatch(publicFavoritesActions.UPDATE_PUBLIC_FAVORITES(results));
            });
    };

export const getIsAdmin = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const resource = getState().auth.user!.isAdmin;
        return resource;
    };
