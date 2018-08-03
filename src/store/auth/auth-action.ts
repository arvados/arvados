// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, default as unionize, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { User } from "../../models/user";
import { RootState } from "../store";
import { ServiceRepository } from "../../services/services";
import { AxiosInstance } from "axios";

export const authActions = unionize({
    SAVE_API_TOKEN: ofType<string>(),
    LOGIN: {},
    LOGOUT: {},
    INIT: ofType<{ user: User, token: string }>(),
    USER_DETAILS_REQUEST: {},
    USER_DETAILS_SUCCESS: ofType<User>()
}, {
    tag: 'type',
    value: 'payload'
});

function setAuthorizationHeader(client: AxiosInstance, token: string) {
    client.defaults.headers.common = {
        Authorization: `OAuth2 ${token}`
    };
}

function removeAuthorizationHeader(client: AxiosInstance) {
    delete client.defaults.headers.common.Authorization;
}

export const initAuth = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const user = services.authService.getUser();
    const token = services.authService.getApiToken();
    if (token) {
        setAuthorizationHeader(services.apiClient, token);
    }
    if (token && user) {
        dispatch(authActions.INIT({ user, token }));
    }
};

export const saveApiToken = (token: string) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    services.authService.saveApiToken(token);
    setAuthorizationHeader(services.apiClient, token);
    dispatch(authActions.SAVE_API_TOKEN(token));
};

export const login = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    services.authService.login();
    dispatch(authActions.LOGIN());
};

export const logout = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    services.authService.removeApiToken();
    services.authService.removeUser();
    removeAuthorizationHeader(services.apiClient);
    services.authService.logout();
    dispatch(authActions.LOGOUT());
};

export const getUserDetails = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<User> => {
    dispatch(authActions.USER_DETAILS_REQUEST());
    return services.authService.getUserDetails().then(user => {
        services.authService.saveUser(user);
        dispatch(authActions.USER_DETAILS_SUCCESS(user));
        return user;
    });
};

export type AuthAction = UnionOf<typeof authActions>;
