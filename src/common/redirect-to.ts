// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Config } from './config';

const REDIRECT_TO_KEY = 'redirectTo';

export const storeRedirects = () => {
    if (window.location.href.indexOf(REDIRECT_TO_KEY) > -1) {
        const { location: { href }, localStorage } = window;
        const redirectUrl = href.split(`${REDIRECT_TO_KEY}=`)[1];

        if (localStorage) {
            localStorage.setItem(REDIRECT_TO_KEY, redirectUrl);
        }
    }
};

export const handleRedirects = (token: string, config: Config) => {
    const { localStorage } = window;
    const { keepWebServiceUrl } = config;

    if (localStorage && localStorage.getItem(REDIRECT_TO_KEY)) {
        const redirectUrl = localStorage.getItem(REDIRECT_TO_KEY);
        localStorage.removeItem(REDIRECT_TO_KEY);
        if (redirectUrl) {
            const sep = redirectUrl.indexOf("?") > -1 ? "&" : "?";
            window.location.href = `${keepWebServiceUrl}${redirectUrl}${sep}api_token=${token}`;
        }
    }
};
