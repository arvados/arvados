// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { LinkResource } from "~/models/link";
import { AxiosInstance } from "axios";
import { ProgressFn } from "~/services/api/api-progress";

export class LinkService extends CommonResourceService<LinkResource> {
    constructor(serverApi: AxiosInstance, progressFn: ProgressFn) {
        super(serverApi, "links", progressFn);
    }
}
