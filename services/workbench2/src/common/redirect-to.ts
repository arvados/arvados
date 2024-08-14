// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getInlineFileUrl } from 'views-components/context-menu/actions/helpers';
import { Config } from './config';

export const REDIRECT_TO_DOWNLOAD_KEY = 'redirectToDownload';
export const REDIRECT_TO_PREVIEW_KEY = 'redirectToPreview';
export const REDIRECT_TO_KEY = 'redirectTo';

const getRedirectKeyFromUrl = (href: string): string => {
    let params = new URL(href).searchParams;
    switch (true) {
        case params.has(REDIRECT_TO_DOWNLOAD_KEY):
            return REDIRECT_TO_DOWNLOAD_KEY;
        case params.has(REDIRECT_TO_PREVIEW_KEY):
            return REDIRECT_TO_PREVIEW_KEY;
        case params.has(REDIRECT_TO_KEY):
            return REDIRECT_TO_KEY;
        default:
            return "";
    }
}

const getRedirectKeyFromStorage = (localStorage: Storage): string => {
    if (localStorage.getItem(REDIRECT_TO_DOWNLOAD_KEY)) {
        return REDIRECT_TO_DOWNLOAD_KEY;
    } else if (localStorage.getItem(REDIRECT_TO_PREVIEW_KEY)) {
        return REDIRECT_TO_PREVIEW_KEY;
    }
    return "";
}

export const storeRedirects = () => {
    const { location: { href }, localStorage } = window;
    const redirectKey = getRedirectKeyFromUrl(href);

    // Change old redirectTo -> redirectToPreview when storing redirect
    const redirectStoreKey = redirectKey === REDIRECT_TO_KEY ? REDIRECT_TO_PREVIEW_KEY : redirectKey;

    if (localStorage && redirectKey && redirectStoreKey) {
        let params = new URL(href).searchParams;
        localStorage.setItem(redirectStoreKey, params.get(redirectKey) || "");
    }
};

export const handleRedirects = (token: string, config: Config) => {
    const { localStorage } = window;
    const { keepWebServiceUrl, keepWebInlineServiceUrl } = config;

    if (localStorage) {
        const redirectKey = getRedirectKeyFromStorage(localStorage);
        const redirectPath = redirectKey ? localStorage.getItem(redirectKey) : '';
        redirectKey && localStorage.removeItem(redirectKey);

        if (redirectKey && redirectPath) {
            let redirectUrl = new URL(keepWebServiceUrl);
            // encodeURI will not touch characters such as # ? that may be
            // delimiter in overall URL syntax
            // Setting pathname attribute will in effect encode # and ?
            // while leaving others minimally disturbed (useful for debugging
            // and avoids excessive percent-encoding)
            redirectUrl.pathname = encodeURI(redirectPath);
            redirectUrl.searchParams.set("api_token", token);
            let u = redirectUrl.href;
            if (redirectKey === REDIRECT_TO_PREVIEW_KEY) {
                u = getInlineFileUrl(u, keepWebServiceUrl, keepWebInlineServiceUrl);
            }
            if (u) {
                window.location.href = u;
            }
        }
    }
};
