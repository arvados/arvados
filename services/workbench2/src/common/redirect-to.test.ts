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
    // PDH is immaterial; for explanation: md5+sizehint of manifest (replace
    // <LF> with linefeed char)
    // `. d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\040-\040%27?a=b<LF>`
    // ie. empty file with filename 'foo - %27?a=b'
    const underlyingPath = '/c=6b1c735de6ae0f2e60cd75d7de36476f+61/foo - %27?a=b';
    const redirectToParamInput = '/c=6b1c735de6ae0f2e60cd75d7de36476f%2B61/foo%20-%20%2527?a=b';
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
                href: `${location.href}?redirectToDownload=${redirectToParamInput}`,
            } as any;
            Object.defineProperty(mockWindow, 'localStorage', {
                value: {
                    setItem: jest.fn(),
                },
                writable: true
            });
        });

        it('should store decoded target path in the local storage', () => {
            // when
            storeRedirects();

            // then
            expect(mockWindow.localStorage.setItem).toHaveBeenCalledWith('redirectToDownload', underlyingPath);
        });
    });

    describe('handleRedirects', () => {
        beforeEach(() => {
            delete mockWindow.location;
            mockWindow.location = {
                ...locationTemplate,
                href: `${location.href}?redirectToDownload=${redirectToParamInput}`,
            } as any;
            Object.defineProperty(mockWindow, 'localStorage', {
                value: {
                    getItem: () => underlyingPath,
                    removeItem: jest.fn(),
                },
                writable: true
            });
        });

        it('should redirect to page when it is present in local storage', () => {
            // when
            handleRedirects("abcxyz", config);

            // then
            let navTarget = new URL(mockWindow.location.href);
            expect(navTarget.origin).toBe(config.keepWebServiceUrl);
            expect(decodeURIComponent(navTarget.pathname)).toBe(underlyingPath);
            expect(navTarget.search).toBe('?api_token=abcxyz');
        });
    });
});
