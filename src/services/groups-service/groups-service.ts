// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { CommonResourceService, ListResults, ListArguments } from '~/services/common-service/common-resource-service';
import { AxiosInstance } from "axios";
import { CollectionResource } from "~/models/collection";
import { ProjectResource } from "~/models/project";
import { ProcessResource } from "~/models/process";
import { TrashableResource } from "~/models/resource";
import { TrashableResourceService } from "~/services/common-service/trashable-resource-service";
import { GroupResource } from '~/models/group';

export interface ContentsArguments {
    limit?: number;
    offset?: number;
    order?: string;
    filters?: string;
    recursive?: boolean;
    includeTrash?: boolean;
}

export interface SharedArguments extends ListArguments {
    include?: string;
}

export type GroupContentsResource =
    CollectionResource |
    ProjectResource |
    ProcessResource;

export class GroupsService<T extends GroupResource = GroupResource> extends TrashableResourceService<T> {

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

    shared(params: SharedArguments = {}): Promise<ListResults<GroupContentsResource>> {
        return this.serverApi
            .get(this.resourceType + 'shared', { params })
            .then(CommonResourceService.mapResponseKeys);
    }
}

export enum GroupContentsResourcePrefix {
    COLLECTION = "collections",
    PROJECT = "groups",
    PROCESS = "container_requests"
}
