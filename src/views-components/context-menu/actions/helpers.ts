// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const sanitizeToken = (href: string, tokenAsQueryParam = true): string => {
    const [prefix, suffix] = href.split('/t=');
    const [token1, token2, token3, ...rest] = suffix.split('/');
    const token = `${token1}/${token2}/${token3}`;
    const sep = href.indexOf("?") > -1 ? "&" : "?";

    return `${[prefix, ...rest].join('/')}${tokenAsQueryParam ? `${sep}api_token=${token}` : ''}`;
};

export const getClipboardUrl = (href: string, shouldSanitizeToken = true): string => {
    const { origin } = window.location;
    const url = shouldSanitizeToken ? sanitizeToken(href, false) : href;

    return shouldSanitizeToken ? `${origin}?redirectTo=${url}` : `${origin}${url}`;
};
