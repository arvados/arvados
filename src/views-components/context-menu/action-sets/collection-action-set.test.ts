// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionActionSet, readOnlyCollectionActionSet } from "./collection-action-set";

describe('collection-action-set', () => {
    const flattCollectionActionSet = collectionActionSet.reduce((prev, next) => prev.concat(next), []);
    const flattReadOnlyCollectionActionSet = readOnlyCollectionActionSet.reduce((prev, next) => prev.concat(next), []);
    describe('collectionActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattCollectionActionSet.length).toBeGreaterThan(0);
        });

        it('should contain readOnlyCollectionActionSet items', () => {
            // then
            expect(flattCollectionActionSet)
                .toEqual(expect.arrayContaining(flattReadOnlyCollectionActionSet));
        })
    });

    describe('readOnlyCollectionActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattReadOnlyCollectionActionSet.length).toBeGreaterThan(0);
        });

        it('should not contain collectionActionSet items', () => {
            // then
            expect(flattReadOnlyCollectionActionSet)
                .not.toEqual(expect.arrayContaining(flattCollectionActionSet));
        })
    });
});