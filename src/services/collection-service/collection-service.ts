// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { serverApi } from "../../common/api/server-api";
import FilterBuilder from "../../common/api/filter-builder";
import { ArvadosResource } from "../response";
import { Collection } from "../../models/collection";
import { getResourceKind } from "../../models/resource";

interface CollectionResource extends ArvadosResource {
    name: string;
    description: string;
    properties: any;
    portable_data_hash: string;
    manifest_text: string;
    replication_desired: number;
    replication_confirmed: number;
    replication_confirmed_at: string;
    trash_at: string;
    delete_at: string;
    is_trashed: boolean;
}

interface CollectionsResponse {
    offset: number;
    limit: number;
    items: CollectionResource[];
}

export default class CollectionService {
    public getCollectionList = (parentUuid?: string): Promise<Collection[]> => {
        if (parentUuid) {
            const fb = new FilterBuilder();
            fb.addLike("ownerUuid", parentUuid);
            return serverApi.get<CollectionsResponse>('/collections', { params: {
                filters: fb.get()
            }}).then(resp => {
                const collections = resp.data.items.map(g => ({
                    name: g.name,
                    createdAt: g.created_at,
                    modifiedAt: g.modified_at,
                    href: g.href,
                    uuid: g.uuid,
                    ownerUuid: g.owner_uuid,
                    kind: getResourceKind(g.kind)
                } as Collection));
                return collections;
            });
        } else {
            return Promise.resolve([]);
        }
    }
}
