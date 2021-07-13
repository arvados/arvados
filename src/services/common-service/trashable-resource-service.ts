// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { snakeCase } from "lodash";
import { AxiosInstance } from "axios";
import { TrashableResource } from "models/resource";
import { CommonResourceService } from "services/common-service/common-resource-service";
import { ApiActions } from "services/api/api-actions";

export class TrashableResourceService<T extends TrashableResource> extends CommonResourceService<T> {

    constructor(serverApi: AxiosInstance, resourceType: string, actions: ApiActions, readOnlyFields: string[] = []) {
        super(serverApi, resourceType, actions, readOnlyFields);
    }

    trash(uuid: string): Promise<T> {
        return CommonResourceService.defaultResponse(
            this.serverApi
                .post(this.resourceType + `/${uuid}/trash`),
            this.actions
        );
    }

    untrash(uuid: string): Promise<T> {
        const params = {
            ensure_unique_name: true
        };
        return CommonResourceService.defaultResponse(
            this.serverApi
                .post(this.resourceType + `/${uuid}/untrash`, {
                    params: CommonResourceService.mapKeys(snakeCase)(params)
                }),
            this.actions
        );
    }
}
