// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from "store/store";
import { Dispatch } from 'redux';
import { ServiceRepository } from "services/services";
import { showErrorSnackbar } from "store/snackbar/snackbar-actions";
import { updateResources } from "store/resources/resources-actions";
import { authActions } from "store/auth/auth-action";
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { RecentUuid } from "models/user";

export const RECENTLY_VISITED_PANEL_ID = "recentlyVisitedPanel";
const GENERIC_LOAD_ERROR = "Could not load user profile";
const SAVE_RECENT_UUIDS_ERROR = "Could not save recent uuids";

const recentlyVisitedPanelActions = bindDataExplorerActions(RECENTLY_VISITED_PANEL_ID);

export const loadRecentlyVisitedPanel = () => (dispatch: Dispatch) => {
    dispatch(recentlyVisitedPanelActions.REQUEST_ITEMS());
};

export const saveRecentlyVisited = (uuid: string) => async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    const user = getState().auth.user;
    if (user) {
        if (user.uuid !== uuid) {
            const previousRecents = user.prefs?.wb?.recentUuids || [];
            const updatedRecents = updateRecentUuids(previousRecents, uuid);
            const userWithUpdatedRecents = {
                ...user,
                prefs: { ...user.prefs, wb: { ...(user.prefs?.wb || {}), recentUuids: updatedRecents } },
            };
            try {
                const updatedUser = await services.userService.update(user.uuid, userWithUpdatedRecents);
                dispatch(updateResources([updatedUser]));
                // If edited user is current user, update auth store
                const currentUserUuid = getState().auth.user?.uuid;
                if (currentUserUuid && currentUserUuid === updatedUser.uuid) {
                    dispatch(authActions.USER_DETAILS_SUCCESS(updatedUser));
                }
            } catch (e) {
                dispatch(showErrorSnackbar(SAVE_RECENT_UUIDS_ERROR));
            }
        }
    } else {
        dispatch(showErrorSnackbar(GENERIC_LOAD_ERROR));
    }
};

function updateRecentUuids(prevRecents: RecentUuid[], newUuid: string, maxLength = 12): RecentUuid[] {
    const newRecentUuid: RecentUuid = { uuid: newUuid, lastVisited: new Date() };

    if (!prevRecents) {
        return [newRecentUuid];
    }

    const index = prevRecents.findIndex(recent => recent.uuid === newUuid);
    // Remove existing occurrence, if any
    if (index !== -1) {
        prevRecents.splice(index, 1);
    }

    // Add to front
    prevRecents.unshift(newRecentUuid);

    // Enforce max length
    if (prevRecents.length > maxLength) {
        prevRecents.pop();
    }

    return prevRecents;
}
