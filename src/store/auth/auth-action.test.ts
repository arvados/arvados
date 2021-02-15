// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { initAuth } from "./auth-action";
import { API_TOKEN_KEY } from "~/services/auth-service/auth-service";

import 'jest-localstorage-mock';
import { ServiceRepository, createServices } from "~/services/services";
import { configureStore, RootStore } from "../store";
import { createBrowserHistory } from "history";
import { mockConfig } from '~/common/config';
import { ApiActions } from "~/services/api/api-actions";
import { ACCOUNT_LINK_STATUS_KEY } from '~/services/link-account-service/link-account-service';
import Axios from "axios";
import MockAdapter from "axios-mock-adapter";
import { ImportMock } from 'ts-mock-imports';
import * as servicesModule from "~/services/services";

describe('auth-actions', () => {
    const axiosInst = Axios.create({ headers: {} });
    const axiosMock = new MockAdapter(axiosInst);

    let store: RootStore;
    let services: ServiceRepository;
    const config: any = {};
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => { },
        errorFn: (id: string, message: string) => { }
    };
    let importMocks: any[];

    beforeEach(() => {
        axiosMock.reset();
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
        localStorage.clear();
        importMocks = [];
    });

    afterEach(() => {
        importMocks.map(m => m.restore());
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
            });

        axiosMock
            .onGet("https://xc59z.arvadosapi.com/discovery/v1/apis/arvados/v1/rest")
            .reply(200, {
                baseUrl: "https://xc59z.arvadosapi.com/arvados/v1",
                keepWebServiceUrl: "",
                keepWebInlineServiceUrl: "",
                remoteHosts: {},
                rootUrl: "https://xc59z.arvadosapi.com",
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
            rootUrl: "https://zzzzz.arvadosapi.com",
            uuidPrefix: "zzzzz",
            remoteHosts: { xc59z: "xc59z.arvadosapi.com" },
            apiRevision: 12345678,
        };

        store.dispatch(initAuth(config));

        store.subscribe(() => {
            const auth = store.getState().auth;
            if (auth.apiToken === "token" &&
                auth.sessions.length === 2 &&
                auth.sessions[0].status === 2 &&
                auth.sessions[1].status === 2
            ) {
                try {
                    expect(auth).toEqual({
                        apiToken: "token",
                        config: {
                            apiRevision: 12345678,
                            remoteHosts: {
                                "xc59z": "xc59z.arvadosapi.com",
                            },
                            rootUrl: "https://zzzzz.arvadosapi.com",
                            uuidPrefix: "zzzzz",
                        },
                        sshKeys: [],
                        homeCluster: "zzzzz",
                        localCluster: "zzzzz",
                        loginCluster: undefined,
                        remoteHostsConfig: {
                            "zzzzz": {
                                "apiRevision": 12345678,
                                "remoteHosts": {
                                    "xc59z": "xc59z.arvadosapi.com",
                                },
                                "rootUrl": "https://zzzzz.arvadosapi.com",
                                "uuidPrefix": "zzzzz",
                            },
                            "xc59z": mockConfig({
                                apiRevision: 12345678,
                                baseUrl: "https://xc59z.arvadosapi.com/arvados/v1",
                                rootUrl: "https://xc59z.arvadosapi.com",
                                uuidPrefix: "xc59z"
                            })
                        },
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
                            "remoteHost": "xc59z.arvadosapi.com",
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
                    console.log(e);
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
