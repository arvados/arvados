// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { UserResource } from "~/models/user";
import { ProgressFn } from "~/services/api/api-progress";

export class UserService extends CommonResourceService<UserResource> {
    constructor(serverApi: AxiosInstance, progressFn: ProgressFn) {
        super(serverApi, "users", progressFn);
    }
}
