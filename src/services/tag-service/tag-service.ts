// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "../link-service/link-service";
import { LinkResource, LinkClass, TailType } from "../../models/link";
import { FilterBuilder } from "../../common/api/filter-builder";

export class TagService {

    constructor(private linkService: LinkService) { }

    create(uuid: string, data: { key: string; value: string } ) {
        return this.linkService.create({
            headUuid: uuid,
            tailUuid: TailType.COLLECTION,
            linkClass: LinkClass.TAG,
            name: '',
            properties: data
        });
    }

    list(uuid: string) {
        const filters = FilterBuilder
            .create<LinkResource>()
            .addEqual("headUuid", uuid)
            .addEqual("tailUuid", TailType.COLLECTION)
            .addEqual("linkClass", LinkClass.TAG);

        return this.linkService
            .list({ filters })
            .then(results => {
                return results.items;
            });
    }

}