// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { customEncodeURI, encodeHash } from "./url";

export class WebDAV {

    defaults: WebDAVDefaults = {
        baseURL: '',
        headers: {},
    };

    constructor(config?: Partial<WebDAVDefaults>, private createRequest = () => new XMLHttpRequest()) {
        if (config) {
            this.defaults = { ...this.defaults, ...config };
        }
    }

    propfind = (url: string, config: WebDAVRequestConfig = {}) =>
        this.request({
            ...config, url,
            method: 'PROPFIND'
        })

    put = (url: string, data?: any, config: WebDAVRequestConfig = {}) =>
        this.request({
            ...config, url,
            method: 'PUT',
            data
        })

    upload = (url: string, files: File[], config: WebDAVRequestConfig = {}) => {
        return Promise.all(
            files.map(file => this.request({
                ...config, url,
                method: 'PUT',
                data: file
            }))
        );
    }

    copy = (url: string, destination: string, config: WebDAVRequestConfig = {}) =>
        this.request({
            ...config, url,
            method: 'COPY',
            headers: {
                ...config.headers,
                Destination: this.defaults.baseURL
                    ? this.defaults.baseURL.replace(/\/+$/, '') + '/' + destination.replace(/^\/+/, '')
                    : destination
            }
        })

    move = (url: string, destination: string, config: WebDAVRequestConfig = {}) =>
        this.request({
            ...config, url,
            method: 'MOVE',
            headers: {
                ...config.headers,
                Destination: this.defaults.baseURL
                    ? this.defaults.baseURL.replace(/\/+$/, '') + '/' + destination.replace(/^\/+/, '')
                    : destination
            }
        })

    delete = (url: string, config: WebDAVRequestConfig = {}) =>
        this.request({
            ...config, url,
            method: 'DELETE'
        })

    private request = (config: RequestConfig) => {
        return new Promise<XMLHttpRequest>((resolve, reject) => {
            const r = this.createRequest();
            this.defaults.baseURL = this.defaults.baseURL.replace(/\/+$/, '');
            r.open(config.method,
                `${this.defaults.baseURL
                    ? this.defaults.baseURL+'/'
                    : ''}${customEncodeURI(config.url)}`);

            if (config.headers && config.headers.Destination) {
                config.headers.Destination = encodeHash(config.headers.Destination);
            }

            const headers = { ...this.defaults.headers, ...config.headers };
            Object
                .keys(headers)
                .forEach(key => r.setRequestHeader(key, headers[key]));

            if (config.onUploadProgress) {
                r.upload.addEventListener('progress', config.onUploadProgress);
            }

            // This event gets triggered on *any* server response
            r.addEventListener('load', () => {
                if (r.status >= 400) {
                    return reject(r);
                } else {
                    return resolve(r);
                }
            });

            // This event gets triggered on network errors
            r.addEventListener('error', () => {
                return reject(r);
            });

            r.upload.addEventListener('error', () => {
                return reject(r);
            });

            r.send(config.data);
        });
    }
}

export interface WebDAVRequestConfig {
    headers?: {
        [key: string]: string;
    };
    onUploadProgress?: (event: ProgressEvent) => void;
}

interface WebDAVDefaults {
    baseURL: string;
    headers: { [key: string]: string };
}

interface RequestConfig {
    method: string;
    url: string;
    headers?: { [key: string]: string };
    data?: any;
    onUploadProgress?: (event: ProgressEvent) => void;
}