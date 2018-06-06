// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios, { AxiosInstance } from "axios";

export const API_HOST = 'https://qr1hi.arvadosapi.com';

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
