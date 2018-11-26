// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { RepositoryResource } from '~/models/repositories';
import { ApiActions } from '~/services/api/api-actions';

 export class RepositoriesService extends CommonResourceService<RepositoryResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "repositories", actions);
    }

     getAllPermissions() {
        return CommonResourceService.defaultResponse(
            this.serverApi
                .get('repositories/get_all_permissions'),
            this.actions
        );
    }
} 