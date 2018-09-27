// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { snakeCase } from 'lodash';
import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { AxiosInstance } from "axios";
import { ContainerRequestResource } from '~/models/container-request';
import { ApiActions } from "~/services/api/api-actions";

export class ContainerRequestService extends CommonResourceService<ContainerRequestResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "container_requests", actions);
    }

    create(data?: Partial<ContainerRequestResource>) {
        if (data) {
            const { mounts } = data;
            if (mounts) {
                const mappedData = {
                    ...CommonResourceService.mapKeys(snakeCase)(data),
                    mounts,
                };
                return CommonResourceService
                    .defaultResponse(
                        this.serverApi.post<ContainerRequestResource>(this.resourceType, mappedData),
                        this.actions);
            }
        }
        return CommonResourceService
            .defaultResponse(
                this.serverApi
                    .post<ContainerRequestResource>(this.resourceType, data && CommonResourceService.mapKeys(snakeCase)(data)),
                this.actions);
    }
}
