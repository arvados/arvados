// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { LogResource } from 'models/log';
import { CommonResourceService } from "services/common-service/common-resource-service";
import { ApiActions } from "services/api/api-actions";

export class LogService extends CommonResourceService<LogResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "logs", actions);
    }
}
