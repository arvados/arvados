// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { REDIRECT_TO_DOWNLOAD_KEY, REDIRECT_TO_PREVIEW_KEY } from "common/redirect-to";
import { extractUuidKind, ResourceKind } from "models/resource";

export const sanitizeToken = (href: string, tokenAsQueryParam = true): string => {
    const [prefix, suffix] = href.split('/t=');
    const [token1, token2, token3, ...rest] = suffix.split('/');
    const token = `${token1}/${token2}/${token3}`;
    const sep = href.indexOf("?") > -1 ? "&" : "?";

    return `${[prefix, ...rest].join('/')}${tokenAsQueryParam ? `${sep}api_token=${token}` : ''}`;
};

export const replaceCollectionId = (href: string, rep: string): string => {
    const [prefix, suffix] = href.split('/c=');
    const [uuid, ...rest] = suffix.split('/');

    if (rep) {
	return `${prefix}/c=${rep}/${rest.join('/')}`;
    } else {
	return `${prefix}/${rest.join('/')}`;
    }
};

/**
 * @returns A shareable token-free WB2 url that redirects to keep-web after login
 */
export const getCollectionItemClipboardUrl = (href: string, keepWebServiceUrl: string, keepWebInlineServiceUrl: string): string => {
    const url = sanitizeToken(href, false);
    return getInlineFileUrl(url, keepWebServiceUrl, keepWebInlineServiceUrl);
};

export const getInlineFileUrl = (url: string, keepWebSvcUrl: string, keepWebInlineSvcUrl: string): string => {
    const collMatch = url.match(/\/c=([a-z0-9-+]+)\//);
    if (collMatch === null) { return ''; }
    if (extractUuidKind(collMatch[1]) !== ResourceKind.COLLECTION) { return ''; }
    const collId = collMatch[1].replace('+', '-');
    let inlineUrl = keepWebInlineSvcUrl !== ""
        ? url.replace(keepWebSvcUrl, keepWebInlineSvcUrl)
        : url;
    let uuidOnHostname = false;
    // Inline URLs as 'https://*.collections.example.com' or
    // 'https://*--collections.example.com' should get the uuid on their hostnames
    // See: https://doc.arvados.org/v2.1/api/keep-web-urls.html
    if (inlineUrl.indexOf('*.') > -1) {
        inlineUrl = inlineUrl.replace('*.', `${collId}.`);
        uuidOnHostname = true;
    } else if (inlineUrl.indexOf('*--') > -1) {
        inlineUrl = inlineUrl.replace('*--', `${collId}--`);
        uuidOnHostname = true;
    }
    if (uuidOnHostname) {
        inlineUrl = inlineUrl.replace(`/c=${collMatch[1]}`, '');
    }
    return inlineUrl;
};

export const isInlineFileUrlSafe = (url: string, keepWebSvcUrl: string, keepWebInlineSvcUrl: string): boolean => {
  let inlineUrl = keepWebInlineSvcUrl !== ""
      ? url.replace(keepWebSvcUrl, keepWebInlineSvcUrl)
      : url;
  return inlineUrl.indexOf('*.') > -1;
}
