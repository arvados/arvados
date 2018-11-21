// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { SshKeyResource } from '~/models/ssh-key';
import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { ApiActions } from "~/services/api/api-actions";

export class AuthorizedKeysService extends CommonResourceService<SshKeyResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "authorized_keys", actions);
    }
}