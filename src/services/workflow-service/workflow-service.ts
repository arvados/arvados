// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { CommonResourceService } from "services/common-service/common-resource-service";
import { WorkflowResource } from 'models/workflow';
import { ApiActions } from 'services/api/api-actions';
import { LinkService } from 'services/link-service/link-service';
import { FilterBuilder } from 'services/api/filter-builder';
import { LinkClass } from 'models/link';
import { OrderBuilder } from 'services/api/order-builder';

export class WorkflowService extends CommonResourceService<WorkflowResource> {

    private linksService = new LinkService(this.serverApi, this.actions);

    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "workflows", actions);
    }

    async presets(workflowUuid: string) {

        const { items: presetLinks } = await this.linksService.list({
            filters: new FilterBuilder()
                .addEqual('tail_uuid', workflowUuid)
                .addEqual('link_class', LinkClass.PRESET)
                .getFilters()
        });

        const presetUuids = presetLinks.map(link => link.headUuid);

        return this.list({
            filters: new FilterBuilder()
                .addIn('uuid', presetUuids)
                .getFilters(),
            order: new OrderBuilder<WorkflowResource>()
                .addAsc('name')
                .getOrder(),
        });

    }

}
