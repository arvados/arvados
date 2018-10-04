// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { CommonResourceService, ListResults, ListArguments } from '~/services/common-service/common-resource-service';
import { AxiosInstance } from "axios";
import { CollectionResource } from "~/models/collection";
import { ProjectResource } from "~/models/project";
import { ProcessResource } from "~/models/process";
import { ResourceKind } from '~/models/resource';
import { TrashableResourceService } from "~/services/common-service/trashable-resource-service";
import { ApiActions } from "~/services/api/api-actions";
import { GroupResource } from "~/models/group";

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

    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "groups", actions);
    }

    async contents(uuid: string, args: ContentsArguments = {}): Promise<ListResults<GroupContentsResource>> {
        const { filters, order, ...other } = args;
        const params = {
            ...other,
            filters: filters ? `[${filters}]` : undefined,
            order: order ? order : undefined
        };

        const response = await CommonResourceService.customResponse(
            this.serverApi
                .get(this.resourceType + `${uuid}/contents`, {
                    params: CommonResourceService.mapKeys(_.camelCase)(params)
                }),
            this.actions
        );
        const { items, ...res } = response;
        const mappedItems = items.map((item: any) => {
            if (item.kind === ResourceKind.COLLECTION) {
                const { properties } = item;
                return {
                    ...TrashableResourceService.mapKeys(_.camelCase)(item),
                    properties,
                };
            } else {
                return TrashableResourceService.mapKeys(_.camelCase)(item);
            }
        });
        const mappedResponse = { ...TrashableResourceService.mapKeys(_.camelCase)(res) };
        return { ...mappedResponse, items: mappedItems };
    }

    shared(params: SharedArguments = {}): Promise<ListResults<GroupContentsResource>> {
        return CommonResourceService.defaultResponse(
            this.serverApi
                .get(this.resourceType + 'shared', { params }),
            this.actions
        );
    }
}

export enum GroupContentsResourcePrefix {
    COLLECTION = "collections",
    PROJECT = "groups",
    PROCESS = "container_requests"
}
