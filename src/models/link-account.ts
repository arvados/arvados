// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export enum LinkAccountStatus {
    SUCCESS,
    CANCELLED,
    FAILED
}

export enum LinkAccountType {
    ADD_OTHER_LOGIN,
    ADD_LOCAL_TO_REMOTE,
    ACCESS_OTHER_ACCOUNT,
    ACCESS_OTHER_REMOTE_ACCOUNT
}

export interface AccountToLink {
    type: LinkAccountType;
    userUuid: string;
    token: string;
}
