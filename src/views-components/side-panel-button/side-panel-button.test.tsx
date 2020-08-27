// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isProjectTrashed } from './side-panel-button';

describe('<SidePanelButton />', () => {
    describe('isProjectTrashed', () => {
        it('should return false if project is undefined', () => {
            // given
            const proj = undefined;
            const resources = {};

            // when
            const result = isProjectTrashed(proj, resources);

            // then
            expect(result).toBeFalsy();
        });

        it('should return false if parent project is undefined', () => {
            // given
            const proj = {};
            const resources = {};

            // when
            const result = isProjectTrashed(proj, resources);

            // then
            expect(result).toBeFalsy();
        });

        it('should return false for owner', () => {
            // given
            const proj = {
                ownerUuid: 'ce8i5-tpzed-000000000000000',
            };
            const resources = {};

            // when
            const result = isProjectTrashed(proj, resources);

            // then
            expect(result).toBeFalsy();
        });

        it('should return true for trashed', () => {
            // given
            const proj = {
                isTrashed: true,
            };
            const resources = {};

            // when
            const result = isProjectTrashed(proj, resources);

            // then
            expect(result).toBeTruthy();
        });

        it('should return false for undefined parent projects', () => {
            // given
            const proj = {
                ownerUuid: 'ce8i5-j7d0g-000000000000000',
            };
            const resources = {};

            // when
            const result = isProjectTrashed(proj, resources);

            // then
            expect(result).toBeFalsy();
        });
    });
});