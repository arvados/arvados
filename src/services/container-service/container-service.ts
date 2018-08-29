// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "~/common/api/common-resource-service";
import { AxiosInstance } from "axios";
import { ContainerResource } from '../../models/container';

export class ContainerService extends CommonResourceService<ContainerResource> {
    constructor(serverApi: AxiosInstance) {
        super(serverApi, "containers");
    }
}
