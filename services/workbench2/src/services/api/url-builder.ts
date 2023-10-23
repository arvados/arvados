// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export class UrlBuilder {
    private readonly url: string = "";
    private query: string = "";

    constructor(host: string) {
        this.url = host;
    }

    public addParam(param: string, value: string) {
        if (this.query.length === 0) {
            this.query += "?";
        } else {
            this.query += "&";
        }
        this.query += `${param}=${value}`;
        return this;
    }

    public get() {
        return this.url + this.query;
    }
}

export function joinUrls(url0?: string, url1?: string) {
    let u0 = "";
    if (url0) {
        let idx0 = url0.length - 1;
        while (url0[idx0] === '/') { --idx0; }
        u0 = url0.substring(0, idx0 + 1);
    }
    let u1 = "";
    if (url1) {
        let idx1 = 0;
        while (url1[idx1] === '/') { ++idx1; }
        u1 = url1.substring(idx1);
    }
    let url = u0;
    if (u1.length > 0) {
        url += '/';
    }
    url += u1;
    return url;
}
