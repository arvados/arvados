// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { ResourceKind } from "../../models/resource";
import { CollectionResource } from "../../models/collection";
import { RootState } from "../store";
import { ServiceRepository } from "../../services/services";

export const collectionPanelActions = unionize({
    LOAD_COLLECTION: ofType<{ uuid: string, kind: ResourceKind }>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>()
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



