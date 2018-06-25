// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Collection } from "../../models/collection";
import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { collectionService } from "../../services/services";

const actions = unionize({
    CREATE_COLLECTION: ofType<Collection>(),
    REMOVE_COLLECTION: ofType<string>(),
    COLLECTIONS_REQUEST: ofType<any>(),
    COLLECTIONS_SUCCESS: ofType<{ collections: Collection[] }>(),
}, {
    tag: 'type',
    value: 'payload'
});

export const getCollectionList = (parentUuid?: string) => (dispatch: Dispatch): Promise<Collection[]> => {
    dispatch(actions.COLLECTIONS_REQUEST());
    return collectionService.getCollectionList(parentUuid).then(collections => {
        dispatch(actions.COLLECTIONS_SUCCESS({collections}));
        return collections;
    });
};

export type CollectionAction = UnionOf<typeof actions>;
export default actions;
