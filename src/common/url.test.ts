// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { customDecodeURI, customEncodeURI } from './url';

describe('url', () => {
    describe('customDecodeURI', () => {
        it('should decode encoded URI', () => {
            // given
            const path = 'test%23test%2Ftest';
            const expectedResult = 'test#test%2Ftest';

            // when
            const result = customDecodeURI(path);

            // then
            expect(result).toEqual(expectedResult);
        });

        it('ignores non parsable URI and return its original form', () => {
            // given
            const path = 'test/path/with%wrong/sign';

            // when
            const result = customDecodeURI(path);

            // then
            expect(result).toEqual(path);
        });
    });

    describe('customEncodeURI', () => {
        it('should encode URI', () => {
            // given
            const path = 'test#test/test';
            const expectedResult = 'test%23test/test';

            // when
            const result = customEncodeURI(path);

            // then
            expect(result).toEqual(expectedResult);
        });

        it('ignores non encodable URI and return its original form', () => {
            // given
            const path = 22;

            // when
            const result = customEncodeURI(path as any);

            // then
            expect(result).toEqual(path);
        });
    });
});