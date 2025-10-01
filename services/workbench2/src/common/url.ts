// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export function getUrlParameter(search: string, name: string) {
    const safeName = name.replace(/[[]/, '\\[').replace(/[\]]/, '\\]');
    const regex = new RegExp('[\\?&]' + safeName + '=([^&#]*)');
    const results = regex.exec(search);
    return results === null ? '' : decodeURIComponent(results[1].replace(/\+/g, ' '));
}

export function normalizeURLPath(url: string) {
    const u = new URL(url);
    u.pathname = u.pathname.replace(/\/\//, '/');
    if (u.pathname[u.pathname.length - 1] === '/') {
        u.pathname = u.pathname.substring(0, u.pathname.length - 1);
    }
    return u.toString();
}

export const customEncodeURI = (path: string) => {
    try {
        return path.split('/').map(encodeURIComponent).join('/');
    } catch(e) {}

    return path;
};

export const customDecodeURI = (path: string) => {
    try {
        return path.split('%2F').map(decodeURIComponent).join('%2F');
    } catch(e) {}

    return path;
};

export const injectTokenParam = (url: string, token: string): Promise<string> => {
    if (url.length) {
        if (token.length) {
            const originalUrl = new URL(url);

            // Remove leading ? for easier manipulation
            const search = originalUrl.search.replace(/^\?/, '');

            // Everything after ?
            const params = `${search}${originalUrl.hash}`;

            // Since search and hash seems to not normalize anything,
            // we should expect href to always end exactly with both.
            // This sanity check should always pass
            if (originalUrl.href.endsWith(params)) {
                // It seems easier to lop off search/params and inject token
                // instead of handling user:pass schemes
                const baseUrl = originalUrl.href
                    // Trim the params from the URL
                    .substring(0, originalUrl.href.length - params.length)
                    // Remove trailing ?
                    .replace(/\?$/, '');

                // Prepend arvados token to search and construct search string
                const searchWithToken = [`arvados_api_token=${token}`, search]
                    // Remove empty elements from array to prevent extra &s with empty search
                    .filter(e => String(e).trim())
                    .join('&');

                return Promise.resolve(`${baseUrl}?${searchWithToken}${originalUrl.hash}`);
            } else {
                // Original url does not end with search+hash, cannot add token
                console.error("Failed to add token to malformed URL: " + url);
                return Promise.reject("Malformed URL");
            }
        } else {
            return Promise.reject("User token required");
        }
    } else {
        return Promise.reject("URL cannot be empty");
    }
};
