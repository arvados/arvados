// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios, { AxiosInstance } from "axios";

export const API_HOST = process.env.REACT_APP_ARVADOS_API_HOST;

export const authClient: AxiosInstance = Axios.create();
export const apiClient: AxiosInstance = Axios.create();

export function setServerApiAuthorizationHeader(token: string) {
    [authClient, apiClient].forEach(client => {
        client.defaults.headers.common = {
            Authorization: `OAuth2 ${token}`
        };
    });
}

export function removeServerApiAuthorizationHeader() {
    [authClient, apiClient].forEach(client => {
        delete client.defaults.headers.common.Authorization;
    });
}

export const setBaseUrl = (url: string) => {
    authClient.defaults.baseURL = url;
    apiClient.defaults.baseURL = url + "/arvados/v1";
};
