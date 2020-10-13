// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { initialDataExplorer } from '../data-explorer/data-explorer-reducer'
import { getParams } from './search-results-middleware-service'

describe('search-results-middleware', () => {
    describe('getParams', () => {
        it('should use include_old_versions=true when asked', () => {
            const dataExplorer = initialDataExplorer;
            const query = 'Search term is:pastVersion';
            const apiRev = 20201013;
            const params = getParams(dataExplorer, query, apiRev);
            expect(params.includeOldVersions).toBe(true);
        });

        it('should not use include_old_versions=true when not asked', () => {
            const dataExplorer = initialDataExplorer;
            const query = 'Search term';
            const apiRev = 20201013;
            const params = getParams(dataExplorer, query, apiRev);
            expect(params.includeOldVersions).toBe(false);
        });
    })
})