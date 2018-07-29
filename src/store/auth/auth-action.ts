// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, default as unionize, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { User } from "../../models/user";
import { RootState } from "../store";
import { ServiceRepository } from "../../services/services";

export const authActions = unionize({
    SAVE_API_TOKEN: ofType<string>(),
    LOGIN: {},
    LOGOUT: {},
    INIT: {},
    USER_DETAILS_REQUEST: {},
    USER_DETAILS_SUCCESS: ofType<User>()
}, {
    tag: 'type',
    value: 'payload'
});

export const getUserDetails = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<User> => {
    dispatch(authActions.USER_DETAILS_REQUEST());
    return services.authService.getUserDetails().then(details => {
        dispatch(authActions.USER_DETAILS_SUCCESS(details));
        return details;
    });
};

export type AuthAction = UnionOf<typeof authActions>;
