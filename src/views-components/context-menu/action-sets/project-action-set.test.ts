// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { filterGroupActionSet, projectActionSet, readOnlyProjectActionSet } from "./project-action-set";

describe('project-action-set', () => {
    const flattProjectActionSet = projectActionSet.reduce((prev, next) => prev.concat(next), []);
    const flattReadOnlyProjectActionSet = readOnlyProjectActionSet.reduce((prev, next) => prev.concat(next), []);
    const flattFilterGroupActionSet = filterGroupActionSet.reduce((prev, next) => prev.concat(next), []);

    describe('projectActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattProjectActionSet.length).toBeGreaterThan(0);
        });

        it('should contain readOnlyProjectActionSet items', () => {
            // then
            expect(flattProjectActionSet)
                .toEqual(expect.arrayContaining(flattReadOnlyProjectActionSet));
        })
    });

    describe('readOnlyProjectActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattReadOnlyProjectActionSet.length).toBeGreaterThan(0);
        });

        it('should not contain projectActionSet items', () => {
            // then
            expect(flattReadOnlyProjectActionSet)
                .not.toEqual(expect.arrayContaining(flattProjectActionSet));
        })
    });

    describe('filterGroupActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattFilterGroupActionSet.length).toBeGreaterThan(0);
        });

        it('should not contain projectActionSet items', () => {
            // then
            expect(flattFilterGroupActionSet)
                .not.toEqual(expect.arrayContaining(flattProjectActionSet));
        })
    });
});
