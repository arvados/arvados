// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { storeRedirects, handleRedirects } from './redirect-to';

describe('redirect-to', () => {
    const { location } = window;
    const config: any = {
        keepWebServiceUrl: 'http://localhost'
    };
    const redirectTo = '/test123';
    const locationTemplate = {
        hash: '',
        hostname: '',
        origin: '',
        host: '',
        pathname: '',
        port: '80',
        protocol: 'http',
        search: '',
        reload: () => {},
        replace: () => {},
        assign: () => {},
        ancestorOrigins: [],
        href: '',
    };

    afterAll((): void => {
        window.location = location;
    });

    describe('storeRedirects', () => {
        beforeEach(() => {
            delete window.location;
            window.location = {
                ...locationTemplate,
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
                ...locationTemplate,
                href: `${location.href}?redirectTo=${redirectTo}`,
            } as any;;
            Object.defineProperty(window, 'sessionStorage', {
                value: {
                    getItem: () => redirectTo,
                    removeItem: jest.fn(),
                },
                writable: true
            });
        });

        it('should redirect to page when it is present in session storage', () => {
            // when
            handleRedirects(config);

            // then
            expect(window.location.href).toBe(`${config.keepWebServiceUrl}${redirectTo}`);
        });
    });
});