// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios, { AxiosInstance } from "axios";

export const API_HOST = process.env.REACT_APP_ARVADOS_API_HOST;

export const serverApi: AxiosInstance = Axios.create({
    baseURL: API_HOST + '/arvados/v1'
});

export function setServerApiAuthorizationHeader(token: string) {
    serverApi.defaults.headers.common = {
        'Authorization': `OAuth2 ${token}`
    };}

export function removeServerApiAuthorizationHeader() {
    delete serverApi.defaults.headers.common.Authorization;
}

export const setBaseUrl = (url: string) => {
    serverApi.defaults.baseURL = url + "/arvados/v1";
};
