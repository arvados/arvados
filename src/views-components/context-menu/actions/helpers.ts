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

export const getInlineFileUrl = (url: string, keepWebSvcUrl: string, keepWebInlineSvcUrl: string): string => {
    const collUuidMatch = url.match(/\/c=([a-z0-9-]+)\//);
    if (collUuidMatch === null) { return ''; }
    const collUuid = collUuidMatch[1];
    let inlineUrl = keepWebInlineSvcUrl !== ""
        ? url.replace(keepWebSvcUrl, keepWebInlineSvcUrl)
        : url;
    let uuidOnHostname = false;
    // Inline URLs as 'https://*.collections.example.com' or
    // 'https://*--collections.example.com' should get the uuid on their hostnames
    // See: https://doc.arvados.org/v2.1/api/keep-web-urls.html
    if (inlineUrl.indexOf('*.') > -1) {
        inlineUrl = inlineUrl.replace('*.', `${collUuid}.`);
        uuidOnHostname = true;
    } else if (inlineUrl.indexOf('*--') > -1) {
        inlineUrl = inlineUrl.replace('*--', `${collUuid}--`);
        uuidOnHostname = true;
    }
    if (uuidOnHostname) {
        inlineUrl = inlineUrl.replace(`/c=${collUuid}`, '');
    }
    return inlineUrl;
};