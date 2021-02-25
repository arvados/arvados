// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export function getUrlParameter(search: string, name: string) {
    const safeName = name.replace(/[\[]/, '\\[').replace(/[\]]/, '\\]');
    const regex = new RegExp('[\\?&]' + safeName + '=([^&#]*)');
    const results = regex.exec(search);
    return results === null ? '' : decodeURIComponent(results[1].replace(/\+/g, ' '));
}

export function normalizeURLPath(url: string) {
    const u = new URL(url);
    u.pathname = u.pathname.replace(/\/\//, '/');
    if (u.pathname[u.pathname.length - 1] === '/') {
        u.pathname = u.pathname.substr(0, u.pathname.length - 1);
    }
    return u.toString();
}

export const escapeHashIfRequired = (name: string, defaultTransformation?: Function) =>
    name.indexOf('#') > -1 ?
        encodeURIComponent(name) :
        defaultTransformation ?
            defaultTransformation(name) :
            name;