// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { serverApi } from "../../common/api/server-api";
import { Dispatch } from "redux";
import actions from "../../store/collection/collection-action";
import UrlBuilder from "../../common/api/url-builder";
import FilterBuilder, { FilterField } from "../../common/api/filter-builder";
import { ArvadosResource } from "../response";
import { Collection } from "../../models/collection";

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
    public getCollectionList = (parentUuid?: string) => (dispatch: Dispatch): Promise<Collection[]> => {
        dispatch(actions.COLLECTIONS_REQUEST());
        if (parentUuid) {
            const fb = new FilterBuilder();
            fb.addLike(FilterField.OWNER_UUID, parentUuid);
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
                    kind: g.kind
                } as Collection));
                dispatch(actions.COLLECTIONS_SUCCESS({collections}));
                return collections;
            });
        } else {
            dispatch(actions.COLLECTIONS_SUCCESS({collections: []}));
            return Promise.resolve([]);
        }
    }
}
