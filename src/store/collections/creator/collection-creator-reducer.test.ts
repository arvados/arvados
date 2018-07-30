// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionCreationReducer } from "./collection-creator-reducer";
import { collectionCreateActions } from "./collection-creator-action";

describe('collection-reducer', () => {

    it('should open collection creator dialog', () => {
        const initialState = {
            creator: { opened: false, ownerUuid: "" }
        };
        const collection = {
            creator: { opened: true, ownerUuid: "" },
        };

        const state = collectionCreationReducer(initialState, collectionCreateActions.OPEN_COLLECTION_CREATOR(initialState.creator));
        expect(state).toEqual(collection);
    });

    it('should close collection creator dialog', () => {
        const initialState = {
            creator: { opened: true, ownerUuid: "" }
        };
        const collection = {
            creator: { opened: false, ownerUuid: "" },
        };

        const state = collectionCreationReducer(initialState, collectionCreateActions.CLOSE_COLLECTION_CREATOR());
        expect(state).toEqual(collection);
    });

    it('should reset collection creator dialog props', () => {
        const initialState = {
            creator: { opened: true, ownerUuid: "test" }
        };
        const collection = {
            creator: { opened: false, ownerUuid: "" },
        };

        const state = collectionCreationReducer(initialState, collectionCreateActions.CREATE_COLLECTION_SUCCESS());
        expect(state).toEqual(collection);
    });
});