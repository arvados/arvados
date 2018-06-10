// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { API_HOST, serverApi } from "../../common/server-api";
import { User } from "../../models/user";
import { Dispatch } from "redux";
import actions from "../../store/auth/auth-action";

export const API_TOKEN_KEY = 'apiToken';
export const USER_EMAIL_KEY = 'userEmail';
export const USER_FIRST_NAME_KEY = 'userFirstName';
export const USER_LAST_NAME_KEY = 'userLastName';

export interface UserDetailsResponse {
    email: string;
    first_name: string;
    last_name: string;
    is_admin: boolean;
}

export default class AuthService {

    public saveApiToken(token: string) {
        localStorage.setItem(API_TOKEN_KEY, token);
    }

    public removeApiToken() {
        localStorage.removeItem(API_TOKEN_KEY);
    }

    public getApiToken() {
        return localStorage.getItem(API_TOKEN_KEY) || undefined;
    }

    public getUser(): User | undefined {
        const email = localStorage.getItem(USER_EMAIL_KEY);
        const firstName = localStorage.getItem(USER_FIRST_NAME_KEY);
        const lastName = localStorage.getItem(USER_LAST_NAME_KEY);
        return email && firstName && lastName
            ? { email, firstName, lastName }
            : undefined;
    }

    public saveUser(user: User) {
        localStorage.setItem(USER_EMAIL_KEY, user.email);
        localStorage.setItem(USER_FIRST_NAME_KEY, user.firstName);
        localStorage.setItem(USER_LAST_NAME_KEY, user.lastName);
    }

    public removeUser() {
        localStorage.removeItem(USER_EMAIL_KEY);
        localStorage.removeItem(USER_FIRST_NAME_KEY);
        localStorage.removeItem(USER_LAST_NAME_KEY);
    }

    public login() {
        const currentUrl = `${window.location.protocol}//${window.location.host}/token`;
        window.location.assign(`${API_HOST}/login?return_to=${currentUrl}`);
    }

    public logout() {
        const currentUrl = `${window.location.protocol}//${window.location.host}`;
        window.location.assign(`${API_HOST}/logout?return_to=${currentUrl}`);
    }

    public getUserDetails = () => (dispatch: Dispatch) => {
        dispatch(actions.USER_DETAILS_REQUEST());
        serverApi
            .get<UserDetailsResponse>('/users/current')
            .then(resp => {
                dispatch(actions.USER_DETAILS_SUCCESS(resp.data));
            })
            // .catch(err => {
            // });
    };
}
