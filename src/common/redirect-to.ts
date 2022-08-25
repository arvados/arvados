// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getInlineFileUrl } from 'views-components/context-menu/actions/helpers';
import { Config } from './config';

export const REDIRECT_TO_DOWNLOAD_KEY = 'redirectToDownload';
export const REDIRECT_TO_PREVIEW_KEY = 'redirectToPreview';
export const REDIRECT_TO_KEY = 'redirectTo';

const getRedirectKeyFromUrl = (href: string): string | null => {
    switch (true) {
        case href.indexOf(REDIRECT_TO_DOWNLOAD_KEY) > -1:
            return REDIRECT_TO_DOWNLOAD_KEY;
        case href.indexOf(REDIRECT_TO_PREVIEW_KEY) > -1:
            return REDIRECT_TO_PREVIEW_KEY;
        case href.indexOf(`${REDIRECT_TO_KEY}=`) > -1:
            return REDIRECT_TO_KEY;
        default:
            return null;
    }
}

const getRedirectKeyFromStorage = (localStorage: Storage): string | null => {
    if (localStorage.getItem(REDIRECT_TO_DOWNLOAD_KEY)) {
        return REDIRECT_TO_DOWNLOAD_KEY;
    } else if (localStorage.getItem(REDIRECT_TO_PREVIEW_KEY)) {
        return REDIRECT_TO_PREVIEW_KEY;
    }
    return null;
}

export const storeRedirects = () => {
    const { location: { href }, localStorage } = window;
    const redirectKey = getRedirectKeyFromUrl(href);

    // Change old redirectTo -> redirectToPreview when storing redirect
    const redirectStoreKey = redirectKey === REDIRECT_TO_KEY ? REDIRECT_TO_PREVIEW_KEY : redirectKey;

    if (localStorage && redirectKey && redirectStoreKey) {
        localStorage.setItem(redirectStoreKey, href.split(`${redirectKey}=`)[1]);
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
            const sep = redirectPath.indexOf("?") > -1 ? "&" : "?";
            let redirectUrl = `${keepWebServiceUrl}${redirectPath}${sep}api_token=${token}`;
            if (redirectKey === REDIRECT_TO_PREVIEW_KEY) {
                redirectUrl = getInlineFileUrl(redirectUrl, keepWebServiceUrl, keepWebInlineServiceUrl);
            }
            window.location.href = redirectUrl;
        }
    }
};
