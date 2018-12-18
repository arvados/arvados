// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authActions, AuthAction } from "./auth-action";
import { User } from "~/models/user";
import { ServiceRepository } from "~/services/services";
import { SshKeyResource } from '~/models/ssh-key';
import { Session } from "~/models/session";

export interface AuthState {
    user?: User;
    apiToken?: string;
    sshKeys: SshKeyResource[];
    sessions: Session[];
}

const initialState: AuthState = {
    user: undefined,
    apiToken: undefined,
    sshKeys: [],
    sessions: []
};

export const authReducer = (services: ServiceRepository) => (state = initialState, action: AuthAction) => {
    return authActions.match(action, {
        SAVE_API_TOKEN: (token: string) => {
            return {...state, apiToken: token};
        },
        INIT: ({ user, token }) => {
            return { ...state, user, apiToken: token };
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
        SET_SSH_KEYS: (sshKeys: SshKeyResource[]) => {
            return {...state, sshKeys};
        },
        ADD_SSH_KEY: (sshKey: SshKeyResource) => {
            return { ...state, sshKeys: state.sshKeys.concat(sshKey) };
        },
        REMOVE_SSH_KEY: (uuid: string) => {
            return { ...state, sshKeys: state.sshKeys.filter((sshKey) => sshKey.uuid !== uuid )};
        },
        SET_SESSIONS: (sessions: Session[]) => {
            return { ...state, sessions };
        },
        ADD_SESSION: (session: Session) => {
            return { ...state, sessions: state.sessions.concat(session) };
        },
        REMOVE_SESSION: (clusterId: string) => {
            return {
                ...state,
                sessions: state.sessions.filter(
                    session => session.clusterId !== clusterId
                )};
        },
        UPDATE_SESSION: (session: Session) => {
            return {
                ...state,
                sessions: state.sessions.map(
                    s => s.clusterId === session.clusterId ? session : s
                )};
        },
        default: () => state
    });
};
