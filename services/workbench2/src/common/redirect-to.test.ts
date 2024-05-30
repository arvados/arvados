// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { storeRedirects, handleRedirects } from './redirect-to';

describe('redirect-to', () => {
    const mockWindow: { location?: any, localStorage?: any} = window

    const { location } = mockWindow;
    const config: any = {
        keepWebServiceUrl: 'http://localhost',
        keepWebServiceInlineUrl: 'http://localhost-inline'
    };
    const redirectTo = 'c=acbd18db4cc2f85cedef654fccc4a4d8%2B3/foo';
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
        mockWindow.location = location;
    });

    describe('storeRedirects', () => {
        beforeEach(() => {
            delete mockWindow.location;
            mockWindow.location = {
                ...locationTemplate,
                href: `${location.href}?redirectToDownload=${redirectTo}`,
            } as any;
            Object.defineProperty(mockWindow, 'localStorage', {
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
            expect(mockWindow.localStorage.setItem).toHaveBeenCalledWith('redirectToDownload', decodeURIComponent(redirectTo));
        });
    });

    describe('handleRedirects', () => {
        beforeEach(() => {
            delete mockWindow.location;
            mockWindow.location = {
                ...locationTemplate,
                href: `${location.href}?redirectToDownload=${redirectTo}`,
            } as any;;
            Object.defineProperty(mockWindow, 'localStorage', {
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
            expect(mockWindow.location.href).toBe(`${config.keepWebServiceUrl}${redirectTo}?api_token=abcxyz`);
        });
    });
});
