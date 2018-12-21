// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export enum SessionStatus {
    INVALIDATED,
    BEING_VALIDATED,
    VALIDATED
}

export interface Session {
    clusterId: string;
    remoteHost: string;
    baseUrl: string;
    username: string;
    email: string;
    token: string;
    loggedIn: boolean;
    status: SessionStatus;
    active: boolean;
}
