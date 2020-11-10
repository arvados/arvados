// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sanitizeToken, getClipboardUrl } from "./helpers";

describe('helpers', () => {
    // given
    const url = 'https://example.com/c=zzzzz/t=v2/a/b/LIMS/1.html';

    describe('sanitizeToken', () => {
        it('should sanitize token from the url', () => {
            // when
            const result = sanitizeToken(url);

            // then
            expect(result).toBe('https://example.com/c=zzzzz/LIMS/1.html?api_token=v2/a/b');
        });
    });

    describe('getClipboardUrl', () => {
        it('should add redirectTo query param', () => {
            // when
            const result = getClipboardUrl(url);

            // then
            expect(result).toBe('http://localhost?redirectTo=https://example.com/c=zzzzz/LIMS/1.html');
        });
    });
});