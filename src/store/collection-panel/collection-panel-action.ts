// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { ResourceKind } from "../../models/resource";
import { CollectionResource } from "../../models/collection";
import { RootState } from "../store";
import { ServiceRepository } from "../../services/services";
import { LinkClass, LinkResource } from "../../models/link";

export const collectionPanelActions = unionize({
    LOAD_COLLECTION: ofType<{ uuid: string, kind: ResourceKind }>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>(),
    LOAD_COLLECTION_TAGS: ofType<{ uuid: string }>(),
    LOAD_COLLECTION_TAGS_SUCCESS: ofType<{ tags: LinkResource[] }>(),
    CREATE_COLLECTION_TAG: ofType<{ data: any }>(),
    CREATE_COLLECTION_TAG_SUCCESS: ofType<{ tag: LinkResource }>()
}, { tag: 'type', value: 'payload' });

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

export const loadCollection = (uuid: string, kind: ResourceKind) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(collectionPanelActions.LOAD_COLLECTION({ uuid, kind }));
        return services.collectionService
            .get(uuid)
            .then(item => {
                dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item: item as CollectionResource }));
            });
    };

export const loadCollectionTags = (uuid: string) => 
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(collectionPanelActions.LOAD_COLLECTION_TAGS({ uuid }));
        return services.tagService
            .list(uuid)
            .then(tags => {
                dispatch(collectionPanelActions.LOAD_COLLECTION_TAGS_SUCCESS({ tags }));
            });
    };


export const createCollectionTag = (uuid: string, data: {}) => 
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const linkResource = {
            key: 'testowanie',
            value: 'by Arturo'
        };

        dispatch(collectionPanelActions.CREATE_COLLECTION_TAG({ data: linkResource }));
        return services.tagService
            .create(uuid, linkResource)
            .then(tag => {
                console.log('tag: ', tag);
                dispatch(collectionPanelActions.CREATE_COLLECTION_TAG_SUCCESS({ tag }));
            });
    };
