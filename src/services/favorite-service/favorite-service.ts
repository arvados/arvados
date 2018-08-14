// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "../link-service/link-service";
import { GroupsService, GroupContentsResource } from "../groups-service/groups-service";
import { LinkResource, LinkClass } from "~/models/link";
import { FilterBuilder } from "~/common/api/filter-builder";
import { ListResults } from "~/common/api/common-resource-service";
import { FavoriteOrderBuilder } from "./favorite-order-builder";
import { OrderBuilder } from "~/common/api/order-builder";

export interface FavoriteListArguments {
    limit?: number;
    offset?: number;
    filters?: FilterBuilder;
    order?: FavoriteOrderBuilder;
}

export class FavoriteService {
    constructor(
        private linkService: LinkService,
        private groupsService: GroupsService
    ) { }

    create(data: { userUuid: string; resource: { uuid: string; name: string } }) {
        return this.linkService.create({
            ownerUuid: data.userUuid,
            tailUuid: data.userUuid,
            headUuid: data.resource.uuid,
            linkClass: LinkClass.STAR,
            name: data.resource.name
        });
    }

    delete(data: { userUuid: string; resourceUuid: string; }) {
        return this.linkService
            .list({
                filters: FilterBuilder
                    .create()
                    .addEqual('tailUuid', data.userUuid)
                    .addEqual('headUuid', data.resourceUuid)
                    .addEqual('linkClass', LinkClass.STAR)
            })
            .then(results => Promise.all(
                results.items.map(item => this.linkService.delete(item.uuid))));
    }

    list(userUuid: string, { filters, limit, offset, order }: FavoriteListArguments = {}): Promise<ListResults<GroupContentsResource>> {
        const listFilter = FilterBuilder
            .create()
            .addEqual('tailUuid', userUuid)
            .addEqual('linkClass', LinkClass.STAR);

        return this.linkService
            .list({
                filters: filters ? filters.concat(listFilter) : listFilter,
                limit,
                offset,
                order: order ? order.getLinkOrder() : OrderBuilder.create<LinkResource>()
            })
            .then(results => {
                const uuids = results.items.map(item => item.headUuid);
                return this.groupsService.contents(userUuid, {
                    limit,
                    offset,
                    order: order ? order.getContentOrder() : OrderBuilder.create<GroupContentsResource>(),
                    filters: FilterBuilder.create().addIn('uuid', uuids),
                    recursive: true
                });
            });
    }

    checkPresenceInFavorites(userUuid: string, resourceUuids: string[]): Promise<Record<string, boolean>> {
        return this.linkService
            .list({
                filters: FilterBuilder
                    .create()
                    .addIn("headUuid", resourceUuids)
                    .addEqual("tailUuid", userUuid)
                    .addEqual("linkClass", LinkClass.STAR)
            })
            .then(({ items }) => resourceUuids.reduce((results, uuid) => {
                const isFavorite = items.some(item => item.headUuid === uuid);
                return { ...results, [uuid]: isFavorite };
            }, {}));
    }

}
