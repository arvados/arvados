// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { initialize } from "redux-form";
import { ServiceRepository } from "~/services/services";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { authActions } from "~/store/auth/auth-action";
import { snackbarActions, SnackbarKind } from "~/store/snackbar/snackbar-actions";

export const MY_ACCOUNT_FORM = 'myAccountForm';

export const loadMyAccountPanel = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([{ label: 'User profile'}]));
    };

export const saveEditedUser = (resource: any) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            await services.userService.update(resource.uuid, resource);
            services.authService.saveUser(resource);
            dispatch(authActions.USER_DETAILS_SUCCESS(resource));
            dispatch(initialize(MY_ACCOUNT_FORM, resource));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Profile has been updated.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch(e) {
            return;
        }
    };
