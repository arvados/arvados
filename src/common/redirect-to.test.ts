// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { storeRedirects, handleRedirects } from './redirect-to';

describe('redirect-to', () => {
    const { location } = window;
    const config: any = {
        keepWebServiceUrl: 'http://localhost',
        keepWebServiceInlineUrl: 'http://localhost-inline'
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
        reload: () => { },
        replace: () => { },
        assign: () => { },
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
                href: `${location.href}?redirectToDownload=${redirectTo}`,
            } as any;
            Object.defineProperty(window, 'localStorage', {
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
            expect(window.localStorage.setItem).toHaveBeenCalledWith('redirectToDownload', redirectTo);
        });
    });

    describe('handleRedirects', () => {
        beforeEach(() => {
            delete window.location;
            window.location = {
                ...locationTemplate,
                href: `${location.href}?redirectToDownload=${redirectTo}`,
            } as any;;
            Object.defineProperty(window, 'localStorage', {
                value: {
                    getItem: () => redirectTo,
                    removeItem: jest.fn(),
                },
                writable: true
            });
        });

        it('should redirect to page when it is present in session storage', () => {
            // when
            handleRedirects("abcxyz", config);

            // then
            expect(window.location.href).toBe(`${config.keepWebServiceUrl}${redirectTo}?api_token=abcxyz`);
        });
    });
});
