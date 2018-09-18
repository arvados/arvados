// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { CommonResourceService, ListResults } from "~/services/common-service/common-resource-service";
import { AxiosInstance } from "axios";
import { CollectionResource } from "~/models/collection";
import { ProjectResource } from "~/models/project";
import { ProcessResource } from "~/models/process";
import { TrashableResource } from "~/models/resource";
import { TrashableResourceService } from "~/services/common-service/trashable-resource-service";
import { ApiActions } from "~/services/api/api-actions";

export interface ContentsArguments {
    limit?: number;
    offset?: number;
    order?: string;
    filters?: string;
    recursive?: boolean;
    includeTrash?: boolean;
}

export type GroupContentsResource =
    CollectionResource |
    ProjectResource |
    ProcessResource;

export class GroupsService<T extends TrashableResource = TrashableResource> extends TrashableResourceService<T> {

    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "groups", actions);
    }

    contents(uuid: string, args: ContentsArguments = {}): Promise<ListResults<GroupContentsResource>> {
        const { filters, order, ...other } = args;
        const params = {
            ...other,
            filters: filters ? `[${filters}]` : undefined,
            order: order ? order : undefined
        };
        return CommonResourceService.defaultResponse(
            this.serverApi
                .get(this.resourceType + `${uuid}/contents`, {
                    params: CommonResourceService.mapKeys(_.snakeCase)(params)
                }),
            this.actions
        );
    }
}

export enum GroupContentsResourcePrefix {
    COLLECTION = "collections",
    PROJECT = "groups",
    PROCESS = "container_requests"
}
