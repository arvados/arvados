// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize, UnionOf } from 'common/unionize';
import { Dispatch } from "redux";
import { RootState } from "../store";
import { ServiceRepository } from "services/services";
import { SshKeyResource } from 'models/ssh-key';
import { User } from "models/user";
import { Session } from "models/session";
import { Config } from 'common/config';
import { matchTokenRoute, matchFedTokenRoute } from 'routes/routes';
import { createServices, setAuthorizationHeader } from "services/services";
import { cancelLinking } from 'store/link-account-panel/link-account-panel-actions';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { WORKBENCH_LOADING_SCREEN } from 'store/workbench/workbench-actions';
import { addRemoteConfig, getRemoteHostConfig } from './auth-action-session';
import { getTokenV2 } from 'models/api-client-authorization';

export const authActions = unionize({
    LOGIN: {},
    LOGOUT: ofType<{ deleteLinkData: boolean, preservePath: boolean }>(),
    SET_CONFIG: ofType<{ config: Config }>(),
    SET_EXTRA_TOKEN: ofType<{ extraApiToken: string, extraApiTokenExpiration?: Date }>(),
    RESET_EXTRA_TOKEN: {},
    INIT_USER: ofType<{ user: User, token: string, tokenExpiration?: Date, tokenLocation?: string }>(),
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

export const initAuth = (config: Config) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
    // Cancel any link account ops in progress unless the user has
    // just logged in or there has been a successful link operation
    const data = services.linkAccountService.getLinkOpStatus();
    if (!matchTokenRoute(window.location.pathname) &&
        (!matchFedTokenRoute(window.location.pathname)) && data === undefined) {
        await dispatch<any>(cancelLinking());
    }
    return dispatch<any>(init(config));
};

const init = (config: Config) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const remoteHosts = () => getState().auth.remoteHosts;
    const token = services.authService.getApiToken();
    let homeCluster = services.authService.getHomeCluster();
    if (homeCluster && !config.remoteHosts[homeCluster]) {
        homeCluster = undefined;
    }
    dispatch(authActions.SET_CONFIG({ config }));
    Object.keys(remoteHosts()).forEach((remoteUuid: string) => {
        const remoteHost = remoteHosts()[remoteUuid];
        if (remoteUuid !== config.uuidPrefix) {
            dispatch<any>(addRemoteConfig(remoteHost));
        }
    });
    dispatch(authActions.SET_HOME_CLUSTER(config.loginCluster || homeCluster || config.uuidPrefix));

    if (token && token !== "undefined") {
        dispatch(progressIndicatorActions.START_WORKING(WORKBENCH_LOADING_SCREEN));
        try {
            await dispatch<any>(saveApiToken(token));
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
        }
    }
};

export const getConfig = (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Config => {
    const state = getState().auth;
    return state.remoteHostsConfig[state.localCluster];
};

export const getLocalCluster = (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): string => {
    return getState().auth.localCluster;
};

export const saveApiToken = (token: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
    let config: any;
    const tokenParts = token.split('/');
    const auth = getState().auth;
    config = dispatch<any>(getConfig);

    // If the token is from a LoginCluster federation, get user & token data
    // from the token issuing cluster.
    if (!config) {
        return;
    }
    const lc = (config as Config).loginCluster
    const tokenCluster = tokenParts.length === 3
        ? tokenParts[1].substring(0, 5)
        : undefined;
    if (tokenCluster && tokenCluster !== auth.localCluster &&
        lc && lc === tokenCluster) {
        config = await getRemoteHostConfig(auth.remoteHosts[tokenCluster]);
    }

    const svc = createServices(config, { progressFn: () => { }, errorFn: () => { } });
    setAuthorizationHeader(svc, token);
    try {
        const user = await svc.authService.getUserDetails();
        const client = await svc.apiClientAuthorizationService.get('current');
        const tokenExpiration = client.expiresAt ? new Date(client.expiresAt) : undefined;
        const tokenLocation = await svc.authService.getStorageType();
        dispatch(authActions.INIT_USER({ user, token, tokenExpiration, tokenLocation }));
    } catch (e) {
        dispatch(authActions.LOGOUT({ deleteLinkData: false, preservePath: false }));
    }
};

export const getNewExtraToken = (reuseStored: boolean = false) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const extraToken = getState().auth.extraApiToken;
        if (reuseStored && extraToken !== undefined) {
            const config = dispatch<any>(getConfig);
            const svc = createServices(config, { progressFn: () => { }, errorFn: () => { } });
            setAuthorizationHeader(svc, extraToken);
            try {
                // Check the extra token's validity before using it. Refresh its
                // expiration date just in case it changed.
                const client = await svc.apiClientAuthorizationService.get('current');
                dispatch(authActions.SET_EXTRA_TOKEN({
                    extraApiToken: extraToken,
                    extraApiTokenExpiration: client.expiresAt ? new Date(client.expiresAt) : undefined,
                }));
                return extraToken;
            } catch (e) {
                dispatch(authActions.RESET_EXTRA_TOKEN());
            }
        }
        const user = getState().auth.user;
        const loginCluster = getState().auth.config.clusterConfig.Login.LoginCluster;
        if (user === undefined) { return; }
        if (loginCluster !== "" && getState().auth.homeCluster !== loginCluster) { return; }
        try {
            // Do not show errors on the create call, cluster security configuration may not
            // allow token creation and there's no way to know that from workbench2 side in advance.
            const client = await services.apiClientAuthorizationService.create(undefined, false);
            const newExtraToken = getTokenV2(client);
            dispatch(authActions.SET_EXTRA_TOKEN({
                extraApiToken: newExtraToken,
                extraApiTokenExpiration: client.expiresAt ? new Date(client.expiresAt) : undefined,
            }));
            return newExtraToken;
        } catch {
            console.warn("Cannot create new tokens with the current token, probably because of cluster's security settings.");
            return;
        }
    };

export const login = (uuidPrefix: string, homeCluster: string, loginCluster: string,
    remoteHosts: { [key: string]: string }) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        services.authService.login(uuidPrefix, homeCluster, loginCluster, remoteHosts);
        dispatch(authActions.LOGIN());
    };

export const logout = (deleteLinkData: boolean = false, preservePath: boolean = false) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) =>
        dispatch(authActions.LOGOUT({ deleteLinkData, preservePath }))

export type AuthAction = UnionOf<typeof authActions>;
