// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "../../common/api/common-resource-service";
import { LinkResource } from "../../models/link";
import { AxiosInstance } from "axios";

export class LinkService extends CommonResourceService<LinkResource> {
    constructor(serverApi: AxiosInstance) {
        super(serverApi, "links");
    }
}