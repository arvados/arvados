// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Config } from './config';

const REDIRECT_TO_KEY = 'redirectTo';
export const REDIRECT_TO_APPLY_TO_PATH = 'redirectToApplyToPath';

export const storeRedirects = () => {
    let redirectUrl;
    const { location: { href }, localStorage } = window;
    const applyToPath = href.indexOf(REDIRECT_TO_APPLY_TO_PATH) > -1;

    if (href.indexOf(REDIRECT_TO_KEY) > -1) {
        redirectUrl = href.split(`${REDIRECT_TO_KEY}=`)[1];
    }

    if (localStorage && redirectUrl) {
        localStorage.setItem(REDIRECT_TO_KEY, redirectUrl);

        if (applyToPath) {
            localStorage.setItem(REDIRECT_TO_APPLY_TO_PATH, 'true');
        }
    }
};

export const handleRedirects = (token: string, config: Config) => {
    const { localStorage } = window;
    const { keepWebServiceUrl } = config;

    if (localStorage && localStorage.getItem(REDIRECT_TO_KEY)) {
        const redirectUrl = localStorage.getItem(REDIRECT_TO_KEY);
        localStorage.removeItem(REDIRECT_TO_KEY);
        const applyToPath = localStorage.getItem(REDIRECT_TO_APPLY_TO_PATH);

        if (redirectUrl) {
            if (applyToPath === 'true') {
                localStorage.removeItem(REDIRECT_TO_APPLY_TO_PATH);
                setTimeout(() => {
                    window.location.pathname = redirectUrl;
                }, 0);
            } else {
                const sep = redirectUrl.indexOf("?") > -1 ? "&" : "?";
                window.location.href = `${keepWebServiceUrl}${redirectUrl}${sep}api_token=${token}`;
            }
        }
    }
};
