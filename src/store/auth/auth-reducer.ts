// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authActions, AuthAction } from "./auth-action";
import { User } from "models/user";
import { ServiceRepository } from "services/services";
import { SshKeyResource } from 'models/ssh-key';
import { Session } from "models/session";
import { Config, mockConfig } from 'common/config';

export interface AuthState {
    user?: User;
    apiToken?: string;
    apiTokenExpiration?: Date;
    apiTokenLocation?: string;
    extraApiToken?: string;
    extraApiTokenExpiration?: Date;
    sshKeys: SshKeyResource[];
    sessions: Session[];
    localCluster: string;
    homeCluster: string;
    loginCluster: string;
    remoteHosts: { [key: string]: string };
    remoteHostsConfig: { [key: string]: Config };
    config: Config;
}

const initialState: AuthState = {
    user: undefined,
    apiToken: undefined,
    apiTokenExpiration: undefined,
    apiTokenLocation: undefined,
    extraApiToken: undefined,
    extraApiTokenExpiration: undefined,
    sshKeys: [],
    sessions: [],
    localCluster: "",
    homeCluster: "",
    loginCluster: "",
    remoteHosts: {},
    remoteHostsConfig: {},
    config: mockConfig({})
};

export const authReducer = (services: ServiceRepository) => (state = initialState, action: AuthAction) => {
    return authActions.match(action, {
        SET_CONFIG: ({ config }) =>
            ({
                ...state,
                config,
                localCluster: config.uuidPrefix,
                remoteHosts: {
                    ...config.remoteHosts,
                    [config.uuidPrefix]: new URL(config.rootUrl).host
                },
                homeCluster: config.loginCluster || config.uuidPrefix,
                loginCluster: config.loginCluster,
                remoteHostsConfig: {
                    ...state.remoteHostsConfig,
                    [config.uuidPrefix]: config
                }
            }),
        REMOTE_CLUSTER_CONFIG: ({ config }) =>
            ({
                ...state,
                remoteHostsConfig: {
                    ...state.remoteHostsConfig,
                    [config.uuidPrefix]: config
                },
            }),
        SET_EXTRA_TOKEN: ({ extraApiToken, extraApiTokenExpiration }) =>
            ({ ...state, extraApiToken, extraApiTokenExpiration }),
        RESET_EXTRA_TOKEN: () =>
            ({ ...state, extraApiToken: undefined, extraApiTokenExpiration: undefined }),
        INIT_USER: ({ user, token, tokenExpiration, tokenLocation = state.apiTokenLocation }) =>
            ({ ...state,
                user,
                apiToken: token,
                apiTokenExpiration: tokenExpiration,
                apiTokenLocation: tokenLocation,
                homeCluster: user.uuid.substr(0, 5)
            }),
        LOGIN: () => state,
        LOGOUT: () => ({ ...state, apiToken: undefined }),
        USER_DETAILS_SUCCESS: (user: User) =>
            ({ ...state, user, homeCluster: user.uuid.substr(0, 5) }),
        SET_SSH_KEYS: (sshKeys: SshKeyResource[]) => ({ ...state, sshKeys }),
        ADD_SSH_KEY: (sshKey: SshKeyResource) =>
            ({ ...state, sshKeys: state.sshKeys.concat(sshKey) }),
        REMOVE_SSH_KEY: (uuid: string) =>
            ({ ...state, sshKeys: state.sshKeys.filter((sshKey) => sshKey.uuid !== uuid) }),
        SET_HOME_CLUSTER: (homeCluster: string) => ({ ...state, homeCluster }),
        SET_SESSIONS: (sessions: Session[]) => ({ ...state, sessions }),
        ADD_SESSION: (session: Session) =>
            ({ ...state, sessions: state.sessions.concat(session) }),
        REMOVE_SESSION: (clusterId: string) =>
            ({
                ...state,
                sessions: state.sessions.filter(
                    session => session.clusterId !== clusterId
                )
            }),
        UPDATE_SESSION: (session: Session) =>
            ({
                ...state,
                sessions: state.sessions.map(
                    s => s.clusterId === session.clusterId ? session : s
                )
            }),
        default: () => state
    });
};
