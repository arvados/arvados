// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { RootState } from "store/store";
import { Dispatch } from 'redux';
import { initialize, reset } from "redux-form";
import { ServiceRepository } from "services/services";
import { showErrorSnackbar, showSuccessSnackbar } from "store/snackbar/snackbar-actions";
import { updateResources } from "store/resources/resources-actions";
import { UserResource } from "models/user";
import { authActions } from "store/auth/auth-action";

export const USER_PREFERENCES_FORM = 'userPreferencesForm';

const GENERIC_LOAD_ERROR = "Could not load user profile";
const SAVE_ERROR = "Could not save preferences";

export const loadUserPreferencesPanel = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuid = getState().auth.user?.uuid;
        if (uuid) {
            try {
                const user = await services.userService.get(uuid, false);
                dispatch(initialize(USER_PREFERENCES_FORM, user));
                dispatch(updateResources([user]));
            } catch (e) {
                dispatch(reset(USER_PREFERENCES_FORM));
                dispatch(showErrorSnackbar(GENERIC_LOAD_ERROR));
            }
        } else {
            dispatch(showErrorSnackbar(GENERIC_LOAD_ERROR));
        }
    }

export const saveUserPreferences = (user: UserResource) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        if (user.uuid) {
            try {
                const updatedUser = await services.userService.update(user.uuid, user);
                dispatch(updateResources([updatedUser]));
                // If edited user is current user, update auth store
                const currentUserUuid = getState().auth.user?.uuid;
                if (currentUserUuid && currentUserUuid === updatedUser.uuid) {
                    dispatch(authActions.USER_DETAILS_SUCCESS(updatedUser));
                }
                dispatch(initialize(USER_PREFERENCES_FORM, updatedUser));
                dispatch(showSuccessSnackbar("Preferences saved"));
            } catch (e) {
                dispatch(showErrorSnackbar(SAVE_ERROR));
            }
        } else {
            dispatch(showErrorSnackbar(GENERIC_LOAD_ERROR));
        }
    };
