// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export class WebDAV {
    static create(config?: Partial<WebDAVDefaults>, createRequest?: () => XMLHttpRequest) {
        return new WebDAV(config, createRequest);
    }

    defaults: WebDAVDefaults = {
        baseUrl: '',
        headers: {},
    };

    propfind = (url: string, config: PropfindConfig = {}) =>
        this.request({
            ...config, url,
            method: 'PROPFIND'
        })

    put = (url: string, config: PutConfig = {}) =>
        this.request({
            ...config, url,
            method: 'PUT'
        })

    copy = (url: string, { destination, ...config }: CopyConfig) =>
        this.request({
            ...config, url,
            method: 'COPY',
            headers: { ...config.headers, Destination: this.defaults.baseUrl + destination }
        })

    move = (url: string, { destination, ...config }: MoveConfig) =>
        this.request({
            ...config, url,
            method: 'MOVE',
            headers: { ...config.headers, Destination: this.defaults.baseUrl + destination }
        })

    delete = (url: string, config: DeleteConfig = {}) =>
        this.request({
            ...config, url,
            method: 'DELETE'
        })

    private constructor(config?: Partial<WebDAVDefaults>, private createRequest = () => new XMLHttpRequest()) {
        if (config) {
            this.defaults = { ...this.defaults, ...config };
        }
    }

    private request = (config: RequestConfig) => {
        return new Promise<XMLHttpRequest>((resolve, reject) => {
            const r = this.createRequest();
            r.open(config.method, this.defaults.baseUrl + config.url);

            const headers = { ...this.defaults.headers, ...config.headers };
            Object
                .keys(headers)
                .forEach(key => r.setRequestHeader(key, headers[key]));

            if (config.onProgress) {
                r.addEventListener('progress', config.onProgress);
            }

            r.addEventListener('load', () => resolve(r));
            r.addEventListener('error', () => reject(r));

            r.send(config.data);
        });

    }
}

export interface PropfindConfig extends BaseConfig { }

export interface PutConfig extends BaseConfig {
    data?: any;
    onProgress?: (event: ProgressEvent) => void;
}

export interface CopyConfig extends BaseConfig {
    destination: string;
}

export interface MoveConfig extends BaseConfig {
    destination: string;
}

export interface DeleteConfig extends BaseConfig { }

interface BaseConfig {
    headers?: {
        [key: string]: string;
    };
}

interface WebDAVDefaults {
    baseUrl: string;
    headers: { [key: string]: string };
}

interface RequestConfig {
    method: string;
    url: string;
    headers?: { [key: string]: string };
    data?: any;
    onProgress?: (event: ProgressEvent) => void;
}
