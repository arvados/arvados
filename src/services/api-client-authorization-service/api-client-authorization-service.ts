// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { ApiActions } from '~/services/api/api-actions';
import { ApiClientAuthorization } from '~/models/api-client-authorization';
import { CommonService } from '~/services/common-service/common-service';

export class ApiClientAuthorizationService extends CommonService<ApiClientAuthorization> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "api_client_authorizations", actions);
    }
} 