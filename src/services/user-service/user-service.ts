// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { CommonResourceService } from "services/common-service/common-resource-service";
import { UserResource } from "models/user";
import { ApiActions } from "services/api/api-actions";

export class UserService extends CommonResourceService<UserResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "users", actions);
    }

    activate(uuid: string) {
        return CommonResourceService.defaultResponse(
            this.serverApi
                .post(this.resourceType + `/${uuid}/activate`),
            this.actions
        );
    }

    unsetup(uuid: string) {
        return CommonResourceService.defaultResponse(
            this.serverApi
                .post(this.resourceType + `/${uuid}/unsetup`),
            this.actions
        );
    }
}
