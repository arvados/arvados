// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionActionSet, readOnlyCollectionActionSet } from "./collection-action-set";
import { containsActionSubSet } from "../../../cypress/utils/contains-action-subset";

describe('collection-action-set', () => {
    const flattCollectionActionSet = collectionActionSet.reduce((prev, next) => prev.concat(next), []);
    const flattReadOnlyCollectionActionSet = readOnlyCollectionActionSet.reduce((prev, next) => prev.concat(next), []);
    describe('collectionActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattCollectionActionSet).to.have.length.greaterThan(0);
        });

        it('should contain readOnlyCollectionActionSet items', () => {
            // then
            expect(containsActionSubSet(flattCollectionActionSet, flattReadOnlyCollectionActionSet)).to.be.true;
        })
    });

    describe('readOnlyCollectionActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattReadOnlyCollectionActionSet).to.have.length.greaterThan(0);
        });

        it('should not contain collectionActionSet items', () => {
            // then
            expect(containsActionSubSet(flattReadOnlyCollectionActionSet, flattCollectionActionSet)).to.be.false;
        })
    });
});