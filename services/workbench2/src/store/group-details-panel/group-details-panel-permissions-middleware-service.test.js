// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { initialDataExplorer } from '../data-explorer/data-explorer-reducer'
import { getParams } from './group-details-panel-permissions-middleware-service'

describe('group-details-panel-permissions-middleware', () => {
    describe('getParams', () => {
        it('should paginate', () => {
            // given
            const dataExplorer = initialDataExplorer;
            let params = getParams(dataExplorer, 'uuid');

            // expect
            expect(params.offset).toBe(0);
            expect(params.limit).toBe(50);

            // when
            dataExplorer.page = 1;
            params = getParams(dataExplorer, 'uuid');

            // expect
            expect(params.offset).toBe(50);
            expect(params.limit).toBe(50);
        });
    })
})
