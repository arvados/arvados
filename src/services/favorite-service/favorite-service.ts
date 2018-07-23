// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "../link-service/link-service";
import { GroupsService, GroupContentsResource } from "../groups-service/groups-service";
import { LinkResource, LinkClass } from "../../models/link";
import { FilterBuilder } from "../../common/api/filter-builder";
import { ListArguments, ListResults } from "../../common/api/common-resource-service";
import { OrderBuilder } from "../../common/api/order-builder";

export interface FavoriteListArguments extends ListArguments {
    filters?: FilterBuilder<LinkResource>;
    order?: OrderBuilder<LinkResource>;
}
export class FavoriteService {
    constructor(
        private linkService: LinkService,
        private groupsService: GroupsService
    ) { }

    create(data: { userUuid: string; resourceUuid: string; }) {
        return this.linkService.create({
            ownerUuid: data.userUuid,
            tailUuid: data.userUuid,
            headUuid: data.resourceUuid,
            linkClass: LinkClass.STAR,
            name: data.resourceUuid
        });
    }

    delete(data: { userUuid: string; resourceUuid: string; }) {
        return this.linkService
            .list({
                filters: FilterBuilder
                    .create<LinkResource>()
                    .addEqual('tailUuid', data.userUuid)
                    .addEqual('headUuid', data.resourceUuid)
                    .addEqual('linkClass', LinkClass.STAR)
            })
            .then(results => Promise.all(
                results.items.map(item => this.linkService.delete(item.uuid))));
    }

    list(userUuid: string, args: FavoriteListArguments = {}): Promise<ListResults<GroupContentsResource>> {
        const listFilter = FilterBuilder
            .create<LinkResource>()
            .addEqual('tailUuid', userUuid)
            .addEqual('linkClass', LinkClass.STAR);

        return this.linkService
            .list({
                ...args,
                filters: args.filters ? args.filters.concat(listFilter) : listFilter
            })
            .then(results => {
                const uuids = results.items.map(item => item.headUuid);
                return this.groupsService.contents(userUuid, {
                    limit: args.limit,
                    offset: args.offset,
                    filters: FilterBuilder.create<GroupContentsResource>().addIn('uuid', uuids),
                    recursive: true
                });
            });
    }


}