// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { AxiosInstance } from "axios";
import { TrashableResource } from "src/models/resource";
import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { ProgressFn } from "~/services/api/api-progress";

export class TrashableResourceService<T extends TrashableResource> extends CommonResourceService<T> {

    constructor(serverApi: AxiosInstance, resourceType: string, progressFn: ProgressFn) {
        super(serverApi, resourceType, progressFn);
    }

    trash(uuid: string): Promise<T> {
        return CommonResourceService.defaultResponse(
            this.serverApi
                .post(this.resourceType + `${uuid}/trash`),
            this.progressFn
        );
    }

    untrash(uuid: string): Promise<T> {
        const params = {
            ensure_unique_name: true
        };
        return CommonResourceService.defaultResponse(
            this.serverApi
                .post(this.resourceType + `${uuid}/untrash`, {
                    params: CommonResourceService.mapKeys(_.snakeCase)(params)
                }),
            this.progressFn
        );
    }
}
