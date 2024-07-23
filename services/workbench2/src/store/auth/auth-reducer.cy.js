// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authReducer } from "./auth-reducer";
import { authActions } from "./auth-action";
import { createServices } from "services/services";
import { mockConfig } from 'common/config';

describe('auth-reducer', () => {
    let reducer;
    const actions = {
        progressFn: (id, working) => { },
        errorFn: (id, message) => { }
    };

    before(() => {
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
            isActive: true,
            canWrite: false,
            canManage: false,
        };
        const state = reducer(initialState, authActions.INIT_USER({ user, token: "token" }));
        expect(state).to.deep.equal({
            apiToken: "token",
            apiTokenExpiration: undefined,
            apiTokenLocation: undefined,
            config: mockConfig({}),
            user,
            sshKeys: [],
            sessions: [],
            extraApiToken: undefined,
            extraApiTokenExpiration: undefined,
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
            isActive: true,
            canWrite: false,
            canManage: false,
        };

        const state = reducer(initialState, authActions.USER_DETAILS_SUCCESS(user));
        expect(state).to.deep.equal({
            apiToken: undefined,
            apiTokenExpiration: undefined,
            apiTokenLocation: undefined,
            config: mockConfig({}),
            sshKeys: [],
            sessions: [],
            extraApiToken: undefined,
            extraApiTokenExpiration: undefined,
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
                isActive: true,
                canManage: false,
                canWrite: false,
            }
        });
    });
});
