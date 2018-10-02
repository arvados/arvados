// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { WorkflowResource } from '~/models/workflow';
import { ApiActions } from '~/services/api/api-actions';

export class WorkflowService extends CommonResourceService<WorkflowResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "workflows", actions);
    }
}
