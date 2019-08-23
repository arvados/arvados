// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authReducer, AuthState } from "./auth-reducer";
import { AuthAction, initAuth } from "./auth-action";
import {
    API_TOKEN_KEY,
    USER_EMAIL_KEY,
    USER_FIRST_NAME_KEY,
    USER_LAST_NAME_KEY,
    USER_OWNER_UUID_KEY,
    USER_UUID_KEY,
    USER_IS_ADMIN,
    USER_IS_ACTIVE,
    USER_USERNAME,
    USER_PREFS
} from "~/services/auth-service/auth-service";

import 'jest-localstorage-mock';
import { createServices } from "~/services/services";
import { configureStore, RootStore } from "../store";
import createBrowserHistory from "history/createBrowserHistory";
import { mockConfig } from '~/common/config';
import { ApiActions } from "~/services/api/api-actions";
import { ACCOUNT_LINK_STATUS_KEY} from '~/services/link-account-service/link-account-service';

describe('auth-actions', () => {
    let reducer: (state: AuthState | undefined, action: AuthAction) => any;
    let store: RootStore;
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => { },
        errorFn: (id: string, message: string) => { }
    };

    beforeEach(() => {
        store = configureStore(createBrowserHistory(), createServices(mockConfig({}), actions));
        localStorage.clear();
        reducer = authReducer(createServices(mockConfig({}), actions));
    });

    it('should initialise state with user and api token from local storage', () => {

        // Only test the case when a link account operation is not being cancelled
        sessionStorage.setItem(ACCOUNT_LINK_STATUS_KEY, "0");
        localStorage.setItem(API_TOKEN_KEY, "token");
        localStorage.setItem(USER_EMAIL_KEY, "test@test.com");
        localStorage.setItem(USER_FIRST_NAME_KEY, "John");
        localStorage.setItem(USER_LAST_NAME_KEY, "Doe");
        localStorage.setItem(USER_UUID_KEY, "zzzzz-tpzed-abcefg");
        localStorage.setItem(USER_USERNAME, "username");
        localStorage.setItem(USER_PREFS, JSON.stringify({}));
        localStorage.setItem(USER_OWNER_UUID_KEY, "ownerUuid");
        localStorage.setItem(USER_IS_ADMIN, JSON.stringify(false));
        localStorage.setItem(USER_IS_ACTIVE, JSON.stringify(true));

        const config: any = {
            rootUrl: "https://zzzzz.arvadosapi.com",
            uuidPrefix: "zzzzz",
            remoteHosts: { xc59z: "xc59z.arvadosapi.com" },
        };

        store.dispatch(initAuth(config));

        expect(store.getState().auth).toEqual({
            apiToken: "token",
            sshKeys: [],
            homeCluster: "zzzzz",
            localCluster: "zzzzz",
            remoteHostsConfig: {},
            remoteHosts: {
                zzzzz: "zzzzz.arvadosapi.com",
                xc59z: "xc59z.arvadosapi.com"
            },
            sessions: [{
                "active": true,
                "baseUrl": undefined,
                "clusterId": "zzzzz",
                "email": "test@test.com",
                "loggedIn": true,
                "remoteHost": "https://zzzzz.arvadosapi.com",
                "status": 2,
                "token": "token",
                "username": "John Doe"
            }, {
                "active": false,
                "baseUrl": "",
                "clusterId": "xc59z",
                "email": "",
                "loggedIn": false,
                "remoteHost": "xc59z.arvadosapi.com",
                "status": 0,
                "token": "",
                "username": ""
            }],
            user: {
                email: "test@test.com",
                firstName: "John",
                lastName: "Doe",
                uuid: "zzzzz-tpzed-abcefg",
                ownerUuid: "ownerUuid",
                username: "username",
                prefs: {},
                isAdmin: false,
                isActive: true
            }
        });
    });

    // TODO: Add remaining action tests
    /*
    it('should fire external url to login', () => {
        const initialState = undefined;
        window.location.assign = jest.fn();
        reducer(initialState, authActions.LOGIN());
        expect(window.location.assign).toBeCalledWith(
            `/login?return_to=${window.location.protocol}//${window.location.host}/token`
        );
    });

    it('should fire external url to logout', () => {
        const initialState = undefined;
        window.location.assign = jest.fn();
        reducer(initialState, authActions.LOGOUT());
        expect(window.location.assign).toBeCalledWith(
            `/logout?return_to=${location.protocol}//${location.host}`
        );
    });
    */
});
