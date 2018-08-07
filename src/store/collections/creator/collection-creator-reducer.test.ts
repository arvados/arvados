// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionCreatorReducer } from "./collection-creator-reducer";
import { collectionCreateActions } from "./collection-creator-action";

describe('collection-reducer', () => {

    it('should open collection creator dialog', () => {
        const initialState = { opened: false, ownerUuid: "" };
        const collection = { opened: true, ownerUuid: "" };

        const state = collectionCreatorReducer(initialState, collectionCreateActions.OPEN_COLLECTION_CREATOR(initialState));
        expect(state).toEqual(collection);
    });

    it('should close collection creator dialog', () => {
        const initialState = { opened: true, ownerUuid: "" };
        const collection = { opened: false, ownerUuid: "" };

        const state = collectionCreatorReducer(initialState, collectionCreateActions.CLOSE_COLLECTION_CREATOR());
        expect(state).toEqual(collection);
    });

    it('should reset collection creator dialog props', () => {
        const initialState = { opened: true, ownerUuid: "test" };
        const collection = { opened: false, ownerUuid: "" };

        const state = collectionCreatorReducer(initialState, collectionCreateActions.CREATE_COLLECTION_SUCCESS());
        expect(state).toEqual(collection);
    });
});
