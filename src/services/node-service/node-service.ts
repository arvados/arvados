// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { CommonResourceService } from "services/common-service/common-resource-service";
import { NodeResource } from 'models/node';
import { ApiActions } from 'services/api/api-actions';

export class NodeService extends CommonResourceService<NodeResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "nodes", actions);
    }
} 