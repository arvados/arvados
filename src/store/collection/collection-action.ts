// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Collection } from "../../models/collection";
import { default as unionize, ofType, UnionOf } from "unionize";

const actions = unionize({
    CREATE_COLLECTION: ofType<Collection>(),
    REMOVE_COLLECTION: ofType<string>(),
    COLLECTIONS_REQUEST: ofType<any>(),
    COLLECTIONS_SUCCESS: ofType<{ collections: Collection[] }>(),
}, {
    tag: 'type',
    value: 'payload'
});

export type CollectionAction = UnionOf<typeof actions>;
export default actions;
