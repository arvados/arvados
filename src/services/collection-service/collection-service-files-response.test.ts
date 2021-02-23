// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionFile } from '~/models/collection-file';
import { getFileFullPath } from './collection-service-files-response';

describe('collection-service-files-response', () => {
    describe('getFileFullPath', () => {
        it('should encode weird names', async () => {
            // given
            const file = { 
                name: '#test',
                path: 'http://localhost',
             } as CollectionFile;

            // when
            const result = getFileFullPath(file);

            // then
            expect(result).toBe('http://localhost/#test');
        });

    });
});