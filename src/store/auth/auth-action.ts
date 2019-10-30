// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize, UnionOf } from '~/common/unionize';
import { Dispatch } from "redux";
import { AxiosInstance } from "axios";
import { RootState } from "../store";
import { ServiceRepository } from "~/services/services";
import { SshKeyResource } from '~/models/ssh-key';
import { User, UserResource } from "~/models/user";
import { Session } from "~/models/session";
import { getClusterConfigURL, Config, ClusterConfigJSON, mapRemoteHosts } from '~/common/config';
import { initSessions } from "~/store/auth/auth-action-session";
import { cancelLinking } from '~/store/link-account-panel/link-account-panel-actions';
import { matchTokenRoute, matchFedTokenRoute } from '~/routes/routes';
import Axios from "axios";
import { AxiosError } from "axios";

export const authActions = unionize({
    SAVE_API_TOKEN: ofType<string>(),
    SAVE_USER: ofType<UserResource>(),
    LOGIN: {},
    LOGOUT: {},
    CONFIG: ofType<{ config: Config }>(),
    INIT: ofType<{ user: User, token: string }>(),
    USER_DETAILS_REQUEST: {},
    USER_DETAILS_SUCCESS: ofType<User>(),
    SET_SSH_KEYS: ofType<SshKeyResource[]>(),
    ADD_SSH_KEY: ofType<SshKeyResource>(),
    REMOVE_SSH_KEY: ofType<string>(),
    SET_HOME_CLUSTER: ofType<string>(),
    SET_SESSIONS: ofType<Session[]>(),
    ADD_SESSION: ofType<Session>(),
    REMOVE_SESSION: ofType<string>(),
    UPDATE_SESSION: ofType<Session>(),
    REMOTE_CLUSTER_CONFIG: ofType<{ config: Config }>(),
});

export function setAuthorizationHeader(services: ServiceRepository, token: string) {
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

export const initAuth = (config: Config) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    // Cancel any link account ops in progress unless the user has
    // just logged in or there has been a successful link operation
    const data = services.linkAccountService.getLinkOpStatus();
    if (!matchTokenRoute(location.pathname) && (!matchFedTokenRoute(location.pathname)) && data === undefined) {
        dispatch<any>(cancelLinking()).then(() => {
            dispatch<any>(init(config));
        });
    }
    else {
        dispatch<any>(init(config));
    }
};

const init = (config: Config) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const user = services.authService.getUser();
    const token = services.authService.getApiToken();
    let homeCluster = services.authService.getHomeCluster();
    if (token) {
        setAuthorizationHeader(services, token);
    }
    if (homeCluster && !config.remoteHosts[homeCluster]) {
        homeCluster = undefined;
    }
    dispatch(authActions.CONFIG({ config }));
    dispatch(authActions.SET_HOME_CLUSTER(config.loginCluster || homeCluster || config.uuidPrefix));
    if (token && user) {
        dispatch(authActions.INIT({ user, token }));
        dispatch<any>(initSessions(services.authService, config, user));
        dispatch<any>(getUserDetails()).then((user: User) => {
            dispatch(authActions.INIT({ user, token }));
        }).catch((err: AxiosError) => {
            if (err.response) {
                // Bad token
                if (err.response.status === 401) {
                    logout()(dispatch, getState, services);
                }
            }
        });
    }
    Object.keys(config.remoteHosts).map((k) => {
        Axios.get<ClusterConfigJSON>(getClusterConfigURL(config.remoteHosts[k]))
            .then(response => {
                const remoteConfig = new Config();
                remoteConfig.uuidPrefix = response.data.ClusterID;
                remoteConfig.workbench2Url = response.data.Services.Workbench2.ExternalURL;
                remoteConfig.loginCluster = response.data.Login.LoginCluster;
                mapRemoteHosts(response.data, remoteConfig);
                dispatch(authActions.REMOTE_CLUSTER_CONFIG({ config: remoteConfig }));
            });
    });
};

export const saveApiToken = (token: string) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    services.authService.saveApiToken(token);
    setAuthorizationHeader(services, token);
    dispatch(authActions.SAVE_API_TOKEN(token));
};

export const saveUser = (user: UserResource) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    services.authService.saveUser(user);
    dispatch(authActions.SAVE_USER(user));
};

export const login = (uuidPrefix: string, homeCluster: string, loginCluster: string,
    remoteHosts: { [key: string]: string }) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        services.authService.login(uuidPrefix, homeCluster, loginCluster, remoteHosts);
        dispatch(authActions.LOGIN());
    };

export const logout = (deleteLinkData: boolean = false) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    if (deleteLinkData) {
        services.linkAccountService.removeAccountToLink();
    }
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
