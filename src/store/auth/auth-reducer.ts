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
            services.authService.saveApiToken(token);
            setServerApiAuthorizationHeader(token);
            return {...state, apiToken: token};
        },
        INIT: () => {
            const user = services.authService.getUser();
            const token = services.authService.getApiToken();
            if (token) {
                setServerApiAuthorizationHeader(token);
            }
            return {user, apiToken: token};
        },
        LOGIN: () => {
            services.authService.login();
            return state;
        },
        LOGOUT: () => {
            services.authService.removeApiToken();
            services.authService.removeUser();
            removeServerApiAuthorizationHeader();
            services.authService.logout();
            return {...state, apiToken: undefined};
        },
        USER_DETAILS_SUCCESS: (user: User) => {
            services.authService.saveUser(user);
            return {...state, user};
        },
        default: () => state
    });
};
