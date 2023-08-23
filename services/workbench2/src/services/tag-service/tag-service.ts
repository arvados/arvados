// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "../link-service/link-service";
import { LinkClass } from "models/link";
import { FilterBuilder } from "services/api/filter-builder";
import { TagTailType, TagResource } from "models/tag";
import { OrderBuilder } from "services/api/order-builder";

export class TagService {

    constructor(private linkService: LinkService) { }

    create(uuid: string, data: { key: string; value: string } ) {
        return this.linkService
            .create({
                headUuid: uuid,
                tailUuid: TagTailType.COLLECTION,
                linkClass: LinkClass.TAG,
                name: '',
                properties: data
            })
            .then(tag => tag as TagResource );
    }

    list(uuid: string) {
        const filters = new FilterBuilder()
            .addEqual("head_uuid", uuid)
            .addEqual("tail_uuid", TagTailType.COLLECTION)
            .addEqual("link_class", LinkClass.TAG)
            .getFilters();

        const order = new OrderBuilder<TagResource>()
            .addAsc('createdAt')
            .getOrder();

        return this.linkService
            .list({ filters, order })
            .then(results => {
                return results.items.map((tag => tag as TagResource ));
            });
    }
}
