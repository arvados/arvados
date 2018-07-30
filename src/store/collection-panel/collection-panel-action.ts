// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { CommonResourceService } from "../../common/api/common-resource-service";
import { apiClient } from "../../common/api/server-api";
import { Dispatch } from "redux";
import { ResourceKind } from "../../models/resource";
import { CollectionResource } from "../../models/collection";

export const collectionPanelActions = unionize({
    LOAD_COLLECTION: ofType<{ uuid: string, kind: ResourceKind }>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>(),
}, { tag: 'type', value: 'payload' });

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

export const loadCollection = (uuid: string, kind: ResourceKind) =>
    (dispatch: Dispatch) => {
        dispatch(collectionPanelActions.LOAD_COLLECTION({ uuid, kind }));
        return new CommonResourceService(apiClient, "collections")
            .get(uuid)
            .then(item => {
                dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item: item as CollectionResource }));
            });
    };



