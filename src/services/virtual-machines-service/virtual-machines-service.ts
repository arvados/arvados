// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { CommonResourceService } from "~/services/common-service/common-resource-service";
import { VirtualMachinesResource } from '~/models/virtual-machines';
import { ApiActions } from '~/services/api/api-actions';

export class VirtualMachinesService extends CommonResourceService<VirtualMachinesResource> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "virtual_machines", actions);
    }

    getRequestedDate(): string {
        return localStorage.getItem('requestedDate') || '';
    }

    saveRequestedDate(date: string) {
        localStorage.setItem('requestedDate', date);
    }

    logins(uuid: string) {
        return CommonResourceService.defaultResponse(
            this.serverApi
                .get(`virtual_machines/${uuid}/logins`),
            this.actions
        );
    }

    getAllLogins() {
        return CommonResourceService.defaultResponse(
            this.serverApi
                .get('virtual_machines/get_all_logins'),
            this.actions
        );
    }
}