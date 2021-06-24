// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getNewExtraToken, initAuth } from "./auth-action";
import { API_TOKEN_KEY } from "services/auth-service/auth-service";

import 'jest-localstorage-mock';
import { ServiceRepository, createServices } from "services/services";
import { configureStore, RootStore } from "../store";
import { createBrowserHistory } from "history";
import { mockConfig } from 'common/config';
import { ApiActions } from "services/api/api-actions";
import { ACCOUNT_LINK_STATUS_KEY } from 'services/link-account-service/link-account-service';
import Axios, { AxiosInstance } from "axios";
import MockAdapter from "axios-mock-adapter";
import { ImportMock } from 'ts-mock-imports';
import * as servicesModule from "services/services";
import * as authActionSessionModule from "./auth-action-session";
import { SessionStatus } from "models/session";
import { getRemoteHostConfig } from "./auth-action-session";

describe('auth-actions', () => {
    let axiosInst: AxiosInstance;
    let axiosMock: MockAdapter;

    let store: RootStore;
    let services: ServiceRepository;
    const config: any = {};
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => { },
        errorFn: (id: string, message: string) => { }
    };
    let importMocks: any[];

    beforeEach(() => {
        axiosInst = Axios.create({ headers: {} });
        axiosMock = new MockAdapter(axiosInst);
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
        localStorage.clear();
        importMocks = [];
    });

    afterEach(() => {
        importMocks.map(m => m.restore());
    });

    it('creates an extra token', async () => {
        axiosMock
            .onGet("/users/current")
            .reply(200, {
                email: "test@test.com",
                first_name: "John",
                last_name: "Doe",
                uuid: "zzzzz-tpzed-abcefg",
                owner_uuid: "ownerUuid",
                is_admin: false,
                is_active: true,
                username: "jdoe",
                prefs: {}
            })
            .onGet("/api_client_authorizations/current")
            .reply(200, {
                expires_at: "2140-01-01T00:00:00.000Z",
                api_token: 'extra token',
            })
            .onPost("/api_client_authorizations")
            .replyOnce(200, {
                uuid: 'zzzzz-gj3su-xxxxxxxxxx',
                apiToken: 'extra token',
            })
            .onPost("/api_client_authorizations")
            .reply(200, {
                uuid: 'zzzzz-gj3su-xxxxxxxxxx',
                apiToken: 'extra additional token',
            });

        importMocks.push(ImportMock.mockFunction(servicesModule, 'createServices', services));
        sessionStorage.setItem(ACCOUNT_LINK_STATUS_KEY, "0");
        localStorage.setItem(API_TOKEN_KEY, "token");

        const config: any = {
            rootUrl: "https://zzzzz.example.com",
            uuidPrefix: "zzzzz",
            remoteHosts: { },
            apiRevision: 12345678,
            clusterConfig: {
                Login: { LoginCluster: "" },
            },
        };

        // Set up auth, confirm that no extra token was requested
        await store.dispatch(initAuth(config))
        expect(store.getState().auth.apiToken).toBeDefined();
        expect(store.getState().auth.extraApiToken).toBeUndefined();

        // Ask for an extra token
        await store.dispatch(getNewExtraToken());
        expect(store.getState().auth.apiToken).toBeDefined();
        expect(store.getState().auth.extraApiToken).toBeDefined();
        const extraToken = store.getState().auth.extraApiToken;

        // Ask for a cached extra token
        await store.dispatch(getNewExtraToken(true));
        expect(store.getState().auth.extraApiToken).toBe(extraToken);

        // Ask for a new extra token, should make a second api request
        await store.dispatch(getNewExtraToken(false));
        expect(store.getState().auth.extraApiToken).toBeDefined();
        expect(store.getState().auth.extraApiToken).not.toBe(extraToken);
    });

    it('requests remote token data to login cluster', async () => {
        const localClusterTokenExpiration = "2020-01-01T00:00:00.000Z";
        const loginClusterTokenExpiration = "2140-01-01T00:00:00.000Z";
        axiosMock
            .onGet("/users/current")
            .reply(200, {
                email: "test@test.com",
                first_name: "John",
                last_name: "Doe",
                uuid: "zzzz1-tpzed-abcefg",
                owner_uuid: "ownerUuid",
                is_admin: false,
                is_active: true,
                username: "jdoe",
                prefs: {}
            })
            .onGet("https://zzzz1.example.com/discovery/v1/apis/arvados/v1/rest")
            .reply(200, {
                baseUrl: "https://zzzz1.example.com/arvados/v1",
                keepWebServiceUrl: "",
                keepWebInlineServiceUrl: "",
                remoteHosts: {},
                rootUrl: "https://zzzz1.example.com",
                uuidPrefix: "zzzz1",
                websocketUrl: "",
                workbenchUrl: "",
                workbench2Url: "",
                revision: 12345678
            })
            // Local cluster -- cached token
            .onGet("https://zzzzz.example.com/arvados/v1/api_client_authorizations/current")
            .reply(200, {
                uuid: 'zzzz1-gj3su-aaaaaaa',
                expires_at: localClusterTokenExpiration,
                api_token: 'tokensecret',
            })
            // Login cluster -- authoritative token copy
            .onGet("https://zzzz1.example.com/arvados/v1/api_client_authorizations/current")
            .reply(200, {
                uuid: 'zzzz1-gj3su-aaaaaaa',
                expires_at: loginClusterTokenExpiration,
                api_token: 'tokensecret',
            });

        const config: any = {
            rootUrl: "https://zzzzz.example.com",
            uuidPrefix: "zzzzz",
            remoteHosts: { zzzz1: "zzzz1.example.com" },
            apiRevision: 12345678,
            clusterConfig: {
                Login: { LoginCluster: "zzzz1" },
            },
        };

        const remoteHostConfig = await getRemoteHostConfig(config.remoteHosts.zzzz1, axiosInst);
        expect(remoteHostConfig).not.toBeFalsy;
        services = createServices(remoteHostConfig!, actions, axiosInst);

        importMocks.push(ImportMock.mockFunction(authActionSessionModule, 'getRemoteHostConfig', remoteHostConfig));
        importMocks.push(ImportMock.mockFunction(servicesModule, 'createServices', services));

        sessionStorage.setItem(ACCOUNT_LINK_STATUS_KEY, "0");
        localStorage.setItem(API_TOKEN_KEY, "v2/zzzz1-gj3su-aaaaaaa/tokensecret");

        await store.dispatch(initAuth(config));
        expect(store.getState().auth.apiToken).toBeDefined();
        expect(localClusterTokenExpiration).not.toBe(loginClusterTokenExpiration);
        expect(store.getState().auth.apiTokenExpiration).toEqual(new Date(loginClusterTokenExpiration));
    });

    it('should initialise state with user and api token from local storage', (done) => {
        axiosMock
            .onGet("/users/current")
            .reply(200, {
                email: "test@test.com",
                first_name: "John",
                last_name: "Doe",
                uuid: "zzzzz-tpzed-abcefg",
                owner_uuid: "ownerUuid",
                is_admin: false,
                is_active: true,
                username: "jdoe",
                prefs: {}
            })
            .onGet("/api_client_authorizations/current")
            .reply(200, {
                expires_at: "2140-01-01T00:00:00.000Z"
            })
            .onGet("https://xc59z.example.com/discovery/v1/apis/arvados/v1/rest")
            .reply(200, {
                baseUrl: "https://xc59z.example.com/arvados/v1",
                keepWebServiceUrl: "",
                keepWebInlineServiceUrl: "",
                remoteHosts: {},
                rootUrl: "https://xc59z.example.com",
                uuidPrefix: "xc59z",
                websocketUrl: "",
                workbenchUrl: "",
                workbench2Url: "",
                revision: 12345678
            });

        importMocks.push(ImportMock.mockFunction(servicesModule, 'createServices', services));

        // Only test the case when a link account operation is not being cancelled
        sessionStorage.setItem(ACCOUNT_LINK_STATUS_KEY, "0");
        localStorage.setItem(API_TOKEN_KEY, "token");

        const config: any = {
            rootUrl: "https://zzzzz.example.com",
            uuidPrefix: "zzzzz",
            remoteHosts: { xc59z: "xc59z.example.com" },
            apiRevision: 12345678,
            clusterConfig: {
                Login: { LoginCluster: "" },
            },
        };

        store.dispatch(initAuth(config));

        store.subscribe(() => {
            const auth = store.getState().auth;
            if (auth.apiToken === "token" &&
                auth.sessions.length === 2 &&
                auth.sessions[0].status === SessionStatus.VALIDATED &&
                auth.sessions[1].status === SessionStatus.VALIDATED
            ) {
                try {
                    expect(auth).toEqual({
                        apiToken: "token",
                        apiTokenExpiration: new Date("2140-01-01T00:00:00.000Z"),
                        config: {
                            apiRevision: 12345678,
                            clusterConfig: {
                                Login: {
                                    LoginCluster: "",
                                },
                            },
                            remoteHosts: {
                                "xc59z": "xc59z.example.com",
                            },
                            rootUrl: "https://zzzzz.example.com",
                            uuidPrefix: "zzzzz",
                        },
                        sshKeys: [],
                        extraApiToken: undefined,
                        extraApiTokenExpiration: undefined,
                        homeCluster: "zzzzz",
                        localCluster: "zzzzz",
                        loginCluster: undefined,
                        remoteHostsConfig: {
                            "zzzzz": {
                                "apiRevision": 12345678,
                                "clusterConfig": {
                                    "Login": {
                                        "LoginCluster": "",
                                    },
                                },
                                "remoteHosts": {
                                    "xc59z": "xc59z.example.com",
                                },
                                "rootUrl": "https://zzzzz.example.com",
                                "uuidPrefix": "zzzzz",
                            },
                            "xc59z": mockConfig({
                                apiRevision: 12345678,
                                baseUrl: "https://xc59z.example.com/arvados/v1",
                                rootUrl: "https://xc59z.example.com",
                                uuidPrefix: "xc59z"
                            })
                        },
                        remoteHosts: {
                            zzzzz: "zzzzz.example.com",
                            xc59z: "xc59z.example.com"
                        },
                        sessions: [{
                            "active": true,
                            "baseUrl": undefined,
                            "clusterId": "zzzzz",
                            "email": "test@test.com",
                            "loggedIn": true,
                            "remoteHost": "https://zzzzz.example.com",
                            "status": 2,
                            "token": "token",
                            "name": "John Doe",
                            "apiRevision": 12345678,
                            "uuid": "zzzzz-tpzed-abcefg",
                            "userIsActive": true
                        }, {
                            "active": false,
                            "baseUrl": "",
                            "clusterId": "xc59z",
                            "email": "",
                            "loggedIn": false,
                            "remoteHost": "xc59z.example.com",
                            "status": 2,
                            "token": "",
                            "name": "",
                            "uuid": "",
                            "apiRevision": 0,
                        }],
                        user: {
                            email: "test@test.com",
                            firstName: "John",
                            lastName: "Doe",
                            uuid: "zzzzz-tpzed-abcefg",
                            ownerUuid: "ownerUuid",
                            username: "jdoe",
                            prefs: { profile: {} },
                            isAdmin: false,
                            isActive: true
                        }
                    });
                    done();
                } catch (e) {
                    fail(e);
                }
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
