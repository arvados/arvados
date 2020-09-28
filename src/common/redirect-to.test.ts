// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { storeRedirects, handleRedirects } from './redirect-to';

describe('redirect-to', () => {
    const redirectTo = 'http://localhost/test123';

    describe('storeRedirects', () => {
        beforeEach(() => {
            Object.defineProperty(window, 'location', {
                value: {
                    href: `${window.location.href}?redirectTo=${redirectTo}`
                },
                writable: true
            });
            Object.defineProperty(window, 'sessionStorage', {
                value: {
                    setItem: jest.fn(),
                },
                writable: true
            });
        });

        it('should store redirectTo in the session storage', () => {
            // when
            storeRedirects();

            // then
            expect(window.sessionStorage.setItem).toHaveBeenCalledWith('redirectTo', redirectTo);
        });
    });

    describe('handleRedirects', () => {
        beforeEach(() => {
            Object.defineProperty(window, 'location', {
                value: {
                    href: ''
                },
                writable: true
            });
            Object.defineProperty(window, 'sessionStorage', {
                value: {
                    getItem: () => redirectTo,
                    removeItem: jest.fn(),
                },
                writable: true
            });
        });

        it('should redirect to page when it is present in session storage', () => {
            // given
            const token = 'testToken';

            // when
            handleRedirects(token);

            // then
            expect(window.location.href).toBe(`${redirectTo}?api_token=${token}`);
        });
    });
});