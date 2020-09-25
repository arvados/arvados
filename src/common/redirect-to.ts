// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const REDIRECT_TO_KEY = 'redirectTo';

export const storeRedirects = () => {
    if (window.location.href.indexOf(REDIRECT_TO_KEY) > -1) {
        const { location: { href }, sessionStorage } = window;
        const redirectUrl = href.split(`${REDIRECT_TO_KEY}=`)[1];
        
        if (sessionStorage) {
            sessionStorage.setItem(REDIRECT_TO_KEY, redirectUrl);
        }
    }
};

export const handleRedirects = (token: string) => {
    const { sessionStorage } = window;

    if (sessionStorage && sessionStorage.getItem(REDIRECT_TO_KEY)) {
        const redirectUrl = sessionStorage.getItem(REDIRECT_TO_KEY);
        sessionStorage.removeItem(REDIRECT_TO_KEY);
        window.location.href = `${redirectUrl}?api_token=${token}`;
    }
};