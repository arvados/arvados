// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize, UnionOf } from '~/common/unionize';
import { Dispatch } from "redux";
import { reset, stopSubmit, startSubmit, FormErrors } from 'redux-form';
import { AxiosInstance } from "axios";
import { RootState } from "../store";
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';
import { ServiceRepository } from "~/services/services";
import { getAuthorizedKeysServiceError, AuthorizedKeysServiceError } from '~/services/authorized-keys-service/authorized-keys-service';
import { KeyType, SshKeyResource } from '~/models/ssh-key';
import { User } from "~/models/user";

export const authActions = unionize({
    SAVE_API_TOKEN: ofType<string>(),
    LOGIN: {},
    LOGOUT: {},
    INIT: ofType<{ user: User, token: string }>(),
    USER_DETAILS_REQUEST: {},
    USER_DETAILS_SUCCESS: ofType<User>(),
    SET_SSH_KEYS: ofType<SshKeyResource[]>(),
    ADD_SSH_KEY: ofType<SshKeyResource>(),
    REMOVE_SSH_KEY: ofType<string>()
});

export const SSH_KEY_CREATE_FORM_NAME = 'sshKeyCreateFormName';
export const SSH_KEY_PUBLIC_KEY_DIALOG = 'sshKeyPublicKeyDialog';
export const SSH_KEY_REMOVE_DIALOG = 'sshKeyRemoveDialog';
export const SSH_KEY_ATTRIBUTES_DIALOG = 'sshKeyAttributesDialog';

export interface SshKeyCreateFormDialogData {
    publicKey: string;
    name: string;
}

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

export const openPublicKeyDialog = (name: string, publicKey: string) =>
    dialogActions.OPEN_DIALOG({ id: SSH_KEY_PUBLIC_KEY_DIALOG, data: { name, publicKey } });

export const openSshKeyAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sshKey = getState().auth.sshKeys.find(it => it.uuid === uuid);
        dispatch(dialogActions.OPEN_DIALOG({ id: SSH_KEY_ATTRIBUTES_DIALOG, data: { sshKey } }));
    };

export const openSshKeyRemoveDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: SSH_KEY_REMOVE_DIALOG,
            data: {
                title: 'Remove public key',
                text: 'Are you sure you want to remove this public key?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeSshKey = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...' }));
        await services.authorizedKeysService.delete(uuid);
        dispatch(authActions.REMOVE_SSH_KEY(uuid));
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Public Key has been successfully removed.', hideDuration: 2000 }));
    };

export const createSshKey = (data: SshKeyCreateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getState().auth.user!.uuid;
        const { name, publicKey } = data;
        dispatch(startSubmit(SSH_KEY_CREATE_FORM_NAME));
        try {
            const newSshKey = await services.authorizedKeysService.create({
                name,
                publicKey,
                keyType: KeyType.SSH,
                authorizedUserUuid: userUuid
            });
            dispatch(authActions.ADD_SSH_KEY(newSshKey));
            dispatch(dialogActions.CLOSE_DIALOG({ id: SSH_KEY_CREATE_FORM_NAME }));
            dispatch(reset(SSH_KEY_CREATE_FORM_NAME));
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Public key has been successfully created.",
                hideDuration: 2000
            }));
        } catch (e) {
            const error = getAuthorizedKeysServiceError(e);
            if (error === AuthorizedKeysServiceError.UNIQUE_PUBLIC_KEY) {
                dispatch(stopSubmit(SSH_KEY_CREATE_FORM_NAME, { publicKey: 'Public key already exists.' } as FormErrors));
            } else if (error === AuthorizedKeysServiceError.INVALID_PUBLIC_KEY) {
                dispatch(stopSubmit(SSH_KEY_CREATE_FORM_NAME, { publicKey: 'Public key is invalid' } as FormErrors));
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
