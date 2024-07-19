// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { filterGroupActionSet, projectActionSet, readOnlyProjectActionSet } from "./project-action-set";
import { containsActionSubSet } from "../../../cypress/utils/contains-action-subset";

describe('project-action-set', () => {
    const flattProjectActionSet = projectActionSet.reduce((prev, next) => prev.concat(next), []);
    const flattReadOnlyProjectActionSet = readOnlyProjectActionSet.reduce((prev, next) => prev.concat(next), []);
    const flattFilterGroupActionSet = filterGroupActionSet.reduce((prev, next) => prev.concat(next), []);

    describe('projectActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattProjectActionSet).to.have.length.greaterThan(0);
        });

        it('should contain readOnlyProjectActionSet items', () => {
            // then
            expect(containsActionSubSet(flattProjectActionSet, flattReadOnlyProjectActionSet)).to.be.true;
        })
    });

    describe('readOnlyProjectActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattReadOnlyProjectActionSet).to.have.length.greaterThan(0);
        });

        it('should not contain projectActionSet items', () => {
            // then
            expect(containsActionSubSet(flattReadOnlyProjectActionSet, flattProjectActionSet)).to.be.false;
        })
    });

    describe('filterGroupActionSet', () => {
        it('should not be empty', () => {
            // then
            expect(flattFilterGroupActionSet).to.have.length.greaterThan(0);
        });

        it('should not contain projectActionSet items', () => {
            // then
            expect(containsActionSubSet(flattFilterGroupActionSet, flattProjectActionSet)).to.be.false;
        })
    });
});
