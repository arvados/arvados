// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "services/common-service/common-resource-service";
import { AxiosInstance } from "axios";
import { ContainerRequestResource, ContainerStatus } from 'models/container-request';
import { ApiActions } from "services/api/api-actions";
import { CommonService } from "services/common-service/common-service";

export class ContainerRequestService extends CommonResourceService<ContainerRequestResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "container_requests", actions);
    }

    containerStatus(uuid: string, showErrors?: boolean) {
        return CommonService.defaultResponse(
            this.serverApi
                .get<ContainerStatus>(`/${this.resourceType}/${uuid}/container_status`),
            this.actions,
            true, // mapKeys
            showErrors
        );
    }
}
