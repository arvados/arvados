// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import 'jest-localstorage-mock';
import Axios, { AxiosInstance } from "axios";
import { createBrowserHistory } from "history";

import { authMiddleware } from "./auth-middleware";
import { RootStore, configureStore } from "../store";
import { ServiceRepository, createServices } from "services/services";
import { ApiActions } from "services/api/api-actions";
import { mockConfig } from "common/config";
import { authActions } from "./auth-action";
import { API_TOKEN_KEY } from 'services/auth-service/auth-service';

describe("AuthMiddleware", () => {
    let store: RootStore;
    let services: ServiceRepository;
    let axiosInst: AxiosInstance;
    const config: any = {};
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => { },
        errorFn: (id: string, message: string) => { }
    };

    beforeEach(() => {
        axiosInst = Axios.create({ headers: {} });
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
        localStorage.clear();
    });

    it("handles LOGOUT action", () => {
        localStorage.setItem(API_TOKEN_KEY, 'someToken');
        window.location.assign = jest.fn();
        const next = jest.fn();
        const middleware = authMiddleware(services)(store)(next);
        middleware(authActions.LOGOUT({deleteLinkData: false}));
        expect(window.location.assign).toBeCalledWith(
            `/logout?api_token=someToken&return_to=${location.protocol}//${location.host}`
        );
        expect(localStorage.getItem(API_TOKEN_KEY)).toBeFalsy();
    });
});