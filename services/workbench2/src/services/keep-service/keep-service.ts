// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "services/common-service/common-resource-service";
import { AxiosInstance } from "axios";
import { KeepServiceResource } from "models/keep-services";
import { ApiActions } from "services/api/api-actions";

export class KeepService extends CommonResourceService<KeepServiceResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "keep_services", actions);
    }
}