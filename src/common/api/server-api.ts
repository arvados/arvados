// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";

export function setServerApiAuthorizationHeader(clients: AxiosInstance[], token: string) {
    clients.forEach(client => {
        client.defaults.headers.common = {
            Authorization: `OAuth2 ${token}`
        };
    });
}

export function removeServerApiAuthorizationHeader(clients: AxiosInstance[]) {
    clients.forEach(client => {
        delete client.defaults.headers.common.Authorization;
    });
}
