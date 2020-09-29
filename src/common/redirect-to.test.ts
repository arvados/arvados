// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { storeRedirects, handleRedirects } from './redirect-to';

describe('redirect-to', () => {
    const { location } = window;
    const redirectTo = 'http://localhost/test123';

    afterAll((): void => {
        window.location = location;
    });

    describe('storeRedirects', () => {
        beforeEach(() => {
            delete window.location;
            window.location = {
                href: `${location.href}?redirectTo=${redirectTo}`,
            } as any;
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
            delete window.location;
            window.location = {
                href: `${location.href}?redirectTo=${redirectTo}`,
            } as any;
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