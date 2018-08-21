// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { CommonResourceService, ListResults } from "~/common/api/common-resource-service";
import { AxiosInstance } from "axios";
import { CollectionResource } from "~/models/collection";
import { ProjectResource } from "~/models/project";
import { ProcessResource } from "~/models/process";
import { TrashResource } from "~/models/resource";

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

export class GroupsService<T extends TrashResource = TrashResource> extends CommonResourceService<T> {

    constructor(serverApi: AxiosInstance) {
        super(serverApi, "groups");
    }

    contents(uuid: string, args: ContentsArguments = {}): Promise<ListResults<GroupContentsResource>> {
        const { filters, order, ...other } = args;
        const params = {
            ...other,
            filters: filters ? `[${filters}]` : undefined,
            order: order ? order : undefined
        };
        return this.serverApi
            .get(this.resourceType + `${uuid}/contents`, {
                params: CommonResourceService.mapKeys(_.snakeCase)(params)
            })
            .then(CommonResourceService.mapResponseKeys);
    }

    trash(uuid: string): Promise<T> {
        return this.serverApi
            .post(this.resourceType + `${uuid}/trash`)
            .then(CommonResourceService.mapResponseKeys);
    }

    untrash(uuid: string): Promise<T> {
        const params = {
            ensure_unique_name: true
        };
        return this.serverApi
            .post(this.resourceType + `${uuid}/untrash`, {
                params: CommonResourceService.mapKeys(_.snakeCase)(params)
            })
            .then(CommonResourceService.mapResponseKeys);
    }

}

export enum GroupContentsResourcePrefix {
    COLLECTION = "collections",
    PROJECT = "groups",
    PROCESS = "container_requests"
}
