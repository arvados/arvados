// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "~/common/api/common-resource-service";
import { AxiosInstance } from "axios";
import { LogResource } from '~/models/log';

export class LogService extends CommonResourceService<LogResource> {
    constructor(serverApi: AxiosInstance) {
        super(serverApi, "logs");
    }
}
