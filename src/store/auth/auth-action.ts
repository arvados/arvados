// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize, UnionOf } from '~/common/unionize';
import { Dispatch } from "redux";
import { reset, stopSubmit } from 'redux-form';
import { User } from "~/models/user";
import { RootState } from "../store";
import { ServiceRepository } from "~/services/services";
import { getCommonResourceServiceError, CommonResourceServiceError } from '~/services/common-service/common-resource-service';
import { AxiosInstance } from "axios";
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { SshKeyCreateFormDialogData, SshKey, KeyType } from '~/models/ssh-key';
import { setBreadcrumbs } from '../breadcrumbs/breadcrumbs-actions';

export const authActions = unionize({
    SAVE_API_TOKEN: ofType<string>(),
    LOGIN: {},
    LOGOUT: {},
    INIT: ofType<{ user: User, token: string }>(),
    USER_DETAILS_REQUEST: {},
    USER_DETAILS_SUCCESS: ofType<User>(),
    SET_SSH_KEYS: ofType<SshKey[]>(),
    ADD_SSH_KEY: ofType<SshKey>()
});

export const SSH_KEY_CREATE_FORM_NAME = 'sshKeyCreateFormName';

function setAuthorizationHeader(services: ServiceRepository, token: string) {
    services.apiClient.defaults.headers.common = {
        Authorization: `OAuth2 ${token}`
    };
    services.webdavClient.defaults.headers = {
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
        setAuthorizationHeader(services, token);
    }
    if (token && user) {
        dispatch(authActions.INIT({ user, token }));
    }
};

export const saveApiToken = (token: string) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    services.authService.saveApiToken(token);
    setAuthorizationHeader(services, token);
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

export const openSshKeyCreateDialog = () => dialogActions.OPEN_DIALOG({ id: SSH_KEY_CREATE_FORM_NAME, data: {} });

export const createSshKey = (data: SshKeyCreateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const userUuid = getState().auth.user!.uuid;
            const { name, publicKey } = data;
            const newSshKey = await services.authorizedKeysService.create({
                name, 
                publicKey,
                keyType: KeyType.SSH,
                authorizedUserUuid: userUuid
            });
            dispatch(dialogActions.CLOSE_DIALOG({ id: SSH_KEY_CREATE_FORM_NAME }));
            dispatch(reset(SSH_KEY_CREATE_FORM_NAME));
            dispatch(authActions.ADD_SSH_KEY(newSshKey));
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Public key has been successfully created.",
                hideDuration: 2000
            }));
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_PUBLIC_KEY) {
                dispatch(stopSubmit(SSH_KEY_CREATE_FORM_NAME, { publicKey: 'Public key already exists.' }));
            } else if (error === CommonResourceServiceError.INVALID_PUBLIC_KEY) {
                dispatch(stopSubmit(SSH_KEY_CREATE_FORM_NAME, { publicKey: 'Public key is invalid' }));
            }
        }
    };

export const loadSshKeysPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            dispatch(setBreadcrumbs([{ label: 'SSH Keys'}]));
            const response = await services.authorizedKeysService.list();
            dispatch(authActions.SET_SSH_KEYS(response.items));
        } catch (e) {
            return;
        }
    };


export type AuthAction = UnionOf<typeof authActions>;
