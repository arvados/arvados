// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "../../common/api/common-resource-service";
import { AxiosInstance } from "axios";
import { KeepResource } from "../../models/keep";

export class KeepService extends CommonResourceService<KeepResource> {
    constructor(serverApi: AxiosInstance) {
        super(serverApi, "keep_services");
    }
}
