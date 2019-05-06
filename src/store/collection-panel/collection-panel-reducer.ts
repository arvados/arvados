// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionPanelActions, CollectionPanelAction } from "./collection-panel-action";
import { CollectionResource } from "~/models/collection";

export interface CollectionPanelState {
    item: CollectionResource | null;
    numberOfCollectionsWithSamePDH: number;
}

const initialState = {
    item: null,
    numberOfCollectionsWithSamePDH: 0
};

export const collectionPanelReducer = (state: CollectionPanelState = initialState, action: CollectionPanelAction) =>
    collectionPanelActions.match(action, {
        default: () => state,
        SET_COLLECTION: (item) => ({ ...state, item }),
        LOAD_COLLECTION_SUCCESS: ({ item }) => ({ ...state, item }),
        SET_NUMBER_OF_COLLECTIONS_WITH_SAME_PDH: (num) => ({ ...state, numberOfCollectionsWithSamePDH: num }),
    });
