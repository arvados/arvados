// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import actions, { AuthAction, UserDetailsResponse } from "./auth-action";
import { User } from "../models/user";
import { authService } from "../services/services";
import { removeServerApiAuthorizationHeader, setServerApiAuthorizationHeader } from "../common/server-api";

export interface AuthState {
    user?: User;
    apiToken?: string;
};

const authReducer = (state: AuthState = {}, action: AuthAction) => {
    return actions.match(action, {
        SAVE_API_TOKEN: (token: string) => {
            authService.saveApiToken(token);
            setServerApiAuthorizationHeader(token);
            return {...state, apiToken: token};
        },
        INIT: () => {
            const user = authService.getUser();
            const token = authService.getApiToken();
            return { user, apiToken: token };
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
        USER_DETAILS_SUCCESS: (ud: UserDetailsResponse) => {
            const user = {
                email: ud.email,
                firstName: ud.first_name,
                lastName: ud.last_name
            };
            authService.saveUser(user);
            return {...state, user};
        },
        default: () => state
    });
};

export default authReducer;
