// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getType } from "typesafe-actions";
import actions, { AuthAction } from "./auth-action";
import { User } from "../models/user";
import { authService } from "../services/services";

type AuthState = User | {};

const authReducer = (state: AuthState = {}, action: AuthAction) => {
    switch (action.type) {
        case getType(actions.saveApiToken): {
            authService.saveApiToken(action.payload);
            return {...state, apiToken: action.payload};
        }
        case getType(actions.login): {
            authService.login();
            return state;
        }
        case getType(actions.logout): {
            authService.logout();
            return {...state, apiToken: null };
        }
        default:
            return state;
    }
};

export default authReducer;
