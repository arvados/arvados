// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "../../common/api/common-resource-service";
import { CollectionResource } from "../../models/collection";
import { AxiosInstance } from "axios";

export class CollectionCreationService extends CommonResourceService<CollectionResource> {
    constructor(serverApi: AxiosInstance) {
        super(serverApi, "collections");
    }
}