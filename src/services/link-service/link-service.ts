// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "services/common-service/common-resource-service";
import { LinkResource } from "models/link";
import { AxiosInstance } from "axios";
import { ApiActions } from "services/api/api-actions";

export class LinkService<Resource extends LinkResource = LinkResource> extends CommonResourceService<Resource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "links", actions);
    }
}
