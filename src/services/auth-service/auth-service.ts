// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const API_TOKEN_KEY = 'api_token';
const API_HOST = 'https://qr1hi.arvadosapi.com';

export default class AuthService {

    public saveApiToken(token: string) {
        localStorage.setItem(API_TOKEN_KEY, token);
    }

    public removeApiToken() {
        localStorage.removeItem(API_TOKEN_KEY);
    }

    public getApiToken() {
        return localStorage.getItem(API_TOKEN_KEY);
    }

    public isUserLoggedIn() {
        return this.getApiToken() !== null;
    }

    public login() {
        const currentUrl = `${window.location.protocol}//${window.location.host}/token`;
        window.location.href = `${API_HOST}/login?return_to=${currentUrl}`;
    }

    public logout() {
        this.removeApiToken();
    }
}
