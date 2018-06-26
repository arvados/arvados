// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import actions, { CollectionAction } from "./collection-action";
import { Collection } from "../../models/collection";

export type CollectionState = Collection[];


const collectionsReducer = (state: CollectionState = [], action: CollectionAction) => {
    return actions.match(action, {
        CREATE_COLLECTION: collection => [...state, collection],
        REMOVE_COLLECTION: () => state,
        COLLECTIONS_REQUEST: () => {
            return [];
        },
        COLLECTIONS_SUCCESS: ({ collections }) => {
            return collections;
        },
        default: () => state
    });
};

export default collectionsReducer;
