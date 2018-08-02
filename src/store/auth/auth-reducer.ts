// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authActions, AuthAction } from "./auth-action";
import { User } from "../../models/user";
import { ServiceRepository } from "../../services/services";
import { removeServerApiAuthorizationHeader, setServerApiAuthorizationHeader } from "../../common/api/server-api";

export interface AuthState {
    user?: User;
    apiToken?: string;
}

export const authReducer = (services: ServiceRepository) => (state: AuthState = {}, action: AuthAction) => {
    return authActions.match(action, {
        SAVE_API_TOKEN: (token: string) => {
            return {...state, apiToken: token};
        },
        INIT: ({ user, token }) => {
            return { user, apiToken: token };
        },
        LOGIN: () => {
            return state;
        },
        LOGOUT: () => {
            return {...state, apiToken: undefined};
        },
        USER_DETAILS_SUCCESS: (user: User) => {
            return {...state, user};
        },
        default: () => state
    });
};
