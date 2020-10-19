// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sanitizeToken, getClipboardUrl } from "./helpers";

describe('helpers', () => {
    // given
    const url = 'https://collections.ardev.roche.com/c=ardev-4zz18-k0hamvtwyit6q56/t=v2/arlog-gj3su-stk5unu8570brvs/fryzaq6z1ow1npak5nngldtkoup918isrvlualf134uf1fbtd/LIMS/1.html';

    describe('sanitizeToken', () => {
        it('should sanitize token from the url', () => {
            // when
            const result = sanitizeToken(url);

            // then
            expect(result).toBe('https://collections.ardev.roche.com/c=ardev-4zz18-k0hamvtwyit6q56/LIMS/1.html?api_token=v2/arlog-gj3su-stk5unu8570brvs/fryzaq6z1ow1npak5nngldtkoup918isrvlualf134uf1fbtd');
        });
    });

    describe('getClipboardUrl', () => {
        it('should add redirectTo query param', () => {
            // when
            const result = getClipboardUrl(url);

            // then
            expect(result).toBe('http://localhost?redirectTo=https://collections.ardev.roche.com/c=ardev-4zz18-k0hamvtwyit6q56/LIMS/1.html');
        });
    });
});