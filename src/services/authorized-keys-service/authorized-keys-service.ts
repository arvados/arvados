// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { SshKeyResource } from 'models/ssh-key';
import { CommonResourceService, CommonResourceServiceError } from 'services/common-service/common-resource-service';
import { ApiActions } from "services/api/api-actions";

export enum AuthorizedKeysServiceError {
    UNIQUE_PUBLIC_KEY = 'UniquePublicKey',
    INVALID_PUBLIC_KEY = 'InvalidPublicKey',
}

export class AuthorizedKeysService extends CommonResourceService<SshKeyResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "authorized_keys", actions);
    }
}

export const getAuthorizedKeysServiceError = (errorResponse: any) => {
    if ('errors' in errorResponse && 'errorToken' in errorResponse) {
        const error = errorResponse.errors.join('');
        switch (true) {
            case /Public key does not appear to be a valid ssh-rsa or dsa public key/.test(error):
                return AuthorizedKeysServiceError.INVALID_PUBLIC_KEY;
            case /Public key already exists in the database, use a different key./.test(error):
                return AuthorizedKeysServiceError.UNIQUE_PUBLIC_KEY;
            default:
                return CommonResourceServiceError.UNKNOWN;
        }
    }
    return CommonResourceServiceError.NONE;
};