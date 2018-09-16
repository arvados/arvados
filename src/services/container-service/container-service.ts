// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { AxiosInstance } from "axios";
import { ContainerResource } from '~/models/container';
import { ProgressFn } from "~/services/api/api-progress";

export class ContainerService extends CommonResourceService<ContainerResource> {
    constructor(serverApi: AxiosInstance, progressFn: ProgressFn) {
        super(serverApi, "containers", progressFn);
    }
}
