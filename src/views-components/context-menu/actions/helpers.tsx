// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const sanitizeToken = (href: string, tokenAsQueryParam: boolean = true): string => {
    const [prefix, suffix] = href.split('/t=');
    const [token, ...rest] = suffix.split('/');

    return `${[prefix, ...rest].join('/')}${tokenAsQueryParam ? `?api_token=${token}` : ''}`;
};

export const getClipboardUrl = (href: string): string => {
    const { origin } = window.location;

    return `${origin}?redirectTo=${sanitizeToken(href, false)}`;
};