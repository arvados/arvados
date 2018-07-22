// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authActions, AuthAction } from "./auth-action";
import { User } from "../../models/user";
import { authService } from "../../services/services";
import { removeServerApiAuthorizationHeader, setServerApiAuthorizationHeader } from "../../common/api/server-api";

export interface AuthState {
    user?: User;
    apiToken?: string;
}

export const authReducer = (state: AuthState = {}, action: AuthAction) => {
    return authActions.match(action, {
        SAVE_API_TOKEN: (token: string) => {
            authService.saveApiToken(token);
            setServerApiAuthorizationHeader(token);
            return {...state, apiToken: token};
        },
        INIT: () => {
            const user = authService.getUser();
            const token = authService.getApiToken();
            if (token) {
                setServerApiAuthorizationHeader(token);
            }
            return {user, apiToken: token};
        },
        LOGIN: () => {
            authService.login();
            return state;
        },
        LOGOUT: () => {
            authService.removeApiToken();
            authService.removeUser();
            removeServerApiAuthorizationHeader();
            authService.logout();
            return {...state, apiToken: undefined};
        },
        USER_DETAILS_SUCCESS: (user: User) => {
            authService.saveUser(user);
            return {...state, user};
        },
        default: () => state
    });
};
