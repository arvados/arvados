// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "services/common-service/common-resource-service";
import { AxiosInstance } from "axios";
import { ContainerResource } from 'models/container';
import { ApiActions } from "services/api/api-actions";

export class ContainerService extends CommonResourceService<ContainerResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "containers", actions);
    }
}
