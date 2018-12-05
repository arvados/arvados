// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { initialize } from "redux-form";
import { ServiceRepository } from "~/services/services";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { authActions } from "~/store/auth/auth-action";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";
import { MY_ACCOUNT_FORM } from "~/views/my-account-panel/my-account-panel-root";

export const loadMyAccountPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            dispatch(setBreadcrumbs([{ label: 'User profile'}]));
        } catch (e) {
            return;
        }
    };

export const saveEditedUser = (resource: any) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            await services.userService.update(resource.uuid, resource);
            services.authService.saveUser(resource);
            dispatch(authActions.USER_DETAILS_SUCCESS(resource));
            dispatch(initialize(MY_ACCOUNT_FORM, resource));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Profile has been updated." }));
        } catch(e) {
            return;
        }
    };
