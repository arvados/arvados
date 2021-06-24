// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionPanelActions, CollectionPanelAction } from "./collection-panel-action";
import { CollectionResource } from "models/collection";

export interface CollectionPanelState {
    item: CollectionResource | null;
    loadBigCollections: boolean;
}

const initialState = {
    item: null,
    loadBigCollections: false,
};

export const collectionPanelReducer = (state: CollectionPanelState = initialState, action: CollectionPanelAction) =>
    collectionPanelActions.match(action, {
        default: () => state,
        SET_COLLECTION: (item) => ({
             ...state,
             item,
             loadBigCollections: false,
        }),
        LOAD_COLLECTION_SUCCESS: ({ item }) => ({ ...state, item }),
        LOAD_BIG_COLLECTIONS: (loadBigCollections) => ({ ...state, loadBigCollections}),
    });
