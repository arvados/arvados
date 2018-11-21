// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authActions, AuthAction } from "./auth-action";
import { User } from "~/models/user";
import { ServiceRepository } from "~/services/services";
import { SshKey } from '~/models/ssh-key';

export interface AuthState {
    user?: User;
    apiToken?: string;
    sshKeys?: SshKey[];
}

const initialState: AuthState = {
    user: undefined,
    apiToken: undefined,
    sshKeys: []
};

export const authReducer = (services: ServiceRepository) => (state: AuthState = initialState, action: AuthAction) => {
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
        SET_SSH_KEYS: (sshKeys: SshKey[]) => {
            return {...state, sshKeys};
        },
        ADD_SSH_KEY: (sshKey: SshKey) => {
            return { ...state, sshKeys: state.sshKeys!.concat(sshKey) };
        },
        default: () => state
    });
};
