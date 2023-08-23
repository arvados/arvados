// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { CommonResourceService } from "services/common-service/common-resource-service";
import { UserResource } from "models/user";
import { ApiActions } from "services/api/api-actions";
import { ListResults } from "services/common-service/common-service";

export class UserService extends CommonResourceService<UserResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions, readOnlyFields: string[] = []) {
        super(serverApi, "users", actions, readOnlyFields.concat([
            'fullName',
            'isInvited',
            'writableBy',
        ]));
    }

    activate(uuid: string) {
        return CommonResourceService.defaultResponse<UserResource>(
            this.serverApi
                .post(this.resourceType + `/${uuid}/activate`),
            this.actions
        );
    }

    setup(uuid: string) {
        return CommonResourceService.defaultResponse<ListResults<any>>(
            this.serverApi
                .post(this.resourceType + `/setup`, {}, { params: { uuid } }),
            this.actions
        );
    }

    unsetup(uuid: string) {
        return CommonResourceService.defaultResponse<UserResource>(
            this.serverApi
                .post(this.resourceType + `/${uuid}/unsetup`),
            this.actions
        );
    }
}
