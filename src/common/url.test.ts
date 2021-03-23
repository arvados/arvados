// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { customDecodeURI, customEncodeURI, encodeHash } from './url';

describe('url', () => {
    describe('encodeHash', () => {
        it('should ignore path without hash', () => {
            // given
            const path = 'path/without/hash';

            // when
            const result = encodeHash(path);

            // then
            expect(result).toEqual(path);
        });

        it('should replace all hashes within the path', () => {
            // given
            const path = 'path/with/hash # and one more #';
            const expectedResult = 'path/with/hash %23 and one more %23';

            // when
            const result = encodeHash(path);

            // then
            expect(result).toEqual(expectedResult);
        });
    });

    describe('customEncodeURI', () => {
        it('should decode', () => {
            // given
            const path = 'test%23test%2Ftest';
            const expectedResult = 'test#test/test';

            // when
            const result = customDecodeURI(path);

            // then
            expect(result).toEqual(expectedResult);
        });
    });

    describe('customEncodeURI', () => {
        it('should encode', () => {
            // given
            const path = 'test#test/test';
            const expectedResult = 'test%23test/test';

            // when
            const result = customEncodeURI(path);

            // then
            expect(result).toEqual(expectedResult);
        });
    });
});