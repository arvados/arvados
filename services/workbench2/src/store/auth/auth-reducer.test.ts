// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authReducer, AuthState } from "./auth-reducer";
import { AuthAction, authActions } from "./auth-action";

import 'jest-localstorage-mock';
import { createServices } from "services/services";
import { mockConfig } from 'common/config';
import { ApiActions } from "services/api/api-actions";

describe('auth-reducer', () => {
    let reducer: (state: AuthState | undefined, action: AuthAction) => any;
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => { },
        errorFn: (id: string, message: string) => { }
    };

    beforeAll(() => {
        localStorage.clear();
        reducer = authReducer(createServices(mockConfig({}), actions));
    });

    it('should correctly initialise state', () => {
        const initialState = undefined;
        const user = {
            email: "test@test.com",
            firstName: "John",
            lastName: "Doe",
            uuid: "zzzzz-tpzed-xurymjxw79nv3jz",
            ownerUuid: "ownerUuid",
            username: "username",
            prefs: {},
            isAdmin: false,
            isActive: true
        };
        const state = reducer(initialState, authActions.INIT_USER({ user, token: "token" }));
        expect(state).toEqual({
            apiToken: "token",
            config: mockConfig({}),
            user,
            sshKeys: [],
            sessions: [],
            homeCluster: "zzzzz",
            localCluster: "",
            loginCluster: "",
            remoteHosts: {},
            remoteHostsConfig: {}
        });
    });

    it('should set user details on success fetch', () => {
        const initialState = undefined;

        const user = {
            email: "test@test.com",
            firstName: "John",
            lastName: "Doe",
            uuid: "uuid",
            ownerUuid: "ownerUuid",
            username: "username",
            prefs: {},
            isAdmin: false,
            isActive: true
        };

        const state = reducer(initialState, authActions.USER_DETAILS_SUCCESS(user));
        expect(state).toEqual({
            apiToken: undefined,
            config: mockConfig({}),
            sshKeys: [],
            sessions: [],
            homeCluster: "uuid",
            localCluster: "",
            loginCluster: "",
            remoteHosts: {},
            remoteHostsConfig: {},
            user: {
                email: "test@test.com",
                firstName: "John",
                lastName: "Doe",
                uuid: "uuid",
                ownerUuid: "ownerUuid",
                username: "username",
                prefs: {},
                isAdmin: false,
                isActive: true
            }
        });
    });
});
