// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize, UnionOf } from '~/common/unionize';
import { Dispatch } from "redux";
import { RootState } from "../store";
import { ServiceRepository } from "~/services/services";
import { SshKeyResource } from '~/models/ssh-key';
import { User } from "~/models/user";
import { Session } from "~/models/session";
import { Config } from '~/common/config';
import { matchTokenRoute, matchFedTokenRoute } from '~/routes/routes';
import { createServices, setAuthorizationHeader } from "~/services/services";
import { cancelLinking } from '~/store/link-account-panel/link-account-panel-actions';
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { WORKBENCH_LOADING_SCREEN } from '~/store/workbench/workbench-actions';
import { addRemoteConfig } from './auth-action-session';
import { getTokenV2 } from '~/models/api-client-authorization';

export const authActions = unionize({
    LOGIN: {},
    LOGOUT: ofType<{ deleteLinkData: boolean }>(),
    SET_CONFIG: ofType<{ config: Config }>(),
    SET_EXTRA_TOKEN: ofType<{ extraToken: string }>(),
    INIT_USER: ofType<{ user: User, token: string }>(),
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

export const initAuth = (config: Config) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    // Cancel any link account ops in progress unless the user has
    // just logged in or there has been a successful link operation
    const data = services.linkAccountService.getLinkOpStatus();
    if (!matchTokenRoute(location.pathname) &&
        (!matchFedTokenRoute(location.pathname)) && data === undefined) {
        dispatch<any>(cancelLinking()).then(() => {
            dispatch<any>(init(config));
        });
    } else {
        dispatch<any>(init(config));
    }
};

const init = (config: Config) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
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
        dispatch<any>(saveApiToken(token)).then(() => {
            dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
        }).catch(() => {
            dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
        });
    }
};

export const getConfig = (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Config => {
    const state = getState().auth;
    return state.remoteHostsConfig[state.localCluster];
};

export const saveApiToken = (token: string) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
    const config = dispatch<any>(getConfig);
    const svc = createServices(config, { progressFn: () => { }, errorFn: () => { } });
    setAuthorizationHeader(svc, token);
    return svc.authService.getUserDetails().then((user: User) => {
        dispatch(authActions.INIT_USER({ user, token }));
        // Upon user init, request an extra token that won't be expired on logout
        // for other uses like the "get token" dialog, or S3 URL building.
        dispatch<any>(getNewExtraToken());
    }).catch(() => {
        dispatch(authActions.LOGOUT({ deleteLinkData: false }));
    });
};

export const getNewExtraToken = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user;
        if (user === undefined) { return; }
        try {
            // Do not show errors on the create call, cluster security configuration may not
            // allow token creation and there's no way to know that from workbench2 side in advance.
            const client = await services.apiClientAuthorizationService.create(undefined, false);
            const newExtraToken = getTokenV2(client);
            dispatch(authActions.SET_EXTRA_TOKEN({ extraToken: newExtraToken }));
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

export const logout = (deleteLinkData: boolean = false) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) =>
        dispatch(authActions.LOGOUT({ deleteLinkData }));

export type AuthAction = UnionOf<typeof authActions>;
