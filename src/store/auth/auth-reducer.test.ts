// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import authReducer from "./auth-reducer";
import actions from "./auth-action";
import {
    API_TOKEN_KEY,
    USER_EMAIL_KEY,
    USER_FIRST_NAME_KEY,
    USER_LAST_NAME_KEY
} from "../../services/auth-service/auth-service";
import { API_HOST } from "../../common/api/server-api";

import 'jest-localstorage-mock';

describe('auth-reducer', () => {
    beforeAll(() => {
        localStorage.clear();
    });

    it('should return default state on initialisation', () => {
        const initialState = undefined;
        const state = authReducer(initialState, actions.INIT());
        expect(state).toEqual({
            apiToken: undefined,
            user: undefined
        });
    });

    it('should read user and api token from local storage on init if they are there', () => {
        const initialState = undefined;

        localStorage.setItem(API_TOKEN_KEY, "token");
        localStorage.setItem(USER_EMAIL_KEY, "test@test.com");
        localStorage.setItem(USER_FIRST_NAME_KEY, "John");
        localStorage.setItem(USER_LAST_NAME_KEY, "Doe");

        const state = authReducer(initialState, actions.INIT());
        expect(state).toEqual({
            apiToken: "token",
            user: {
                email: "test@test.com",
                firstName: "John",
                lastName: "Doe"
            }
        });
    });

    it('should store token in local storage', () => {
        const initialState = undefined;

        const state = authReducer(initialState, actions.SAVE_API_TOKEN("token"));
        expect(state).toEqual({
            apiToken: "token",
            user: undefined
        });

        expect(localStorage.getItem(API_TOKEN_KEY)).toBe("token");
    });

    it('should set user details on success fetch', () => {
        const initialState = undefined;

        const userDetails = {
            email: "test@test.com",
            first_name: "John",
            last_name: "Doe",
            is_admin: true
        };

        const state = authReducer(initialState, actions.USER_DETAILS_SUCCESS(userDetails));
        expect(state).toEqual({
            apiToken: undefined,
            user: {
                email: "test@test.com",
                firstName: "John",
                lastName: "Doe"
            }
        });

        expect(localStorage.getItem(API_TOKEN_KEY)).toBe("token");
    });

    it('should fire external url to login', () => {
        const initialState = undefined;
        window.location.assign = jest.fn();
        authReducer(initialState, actions.LOGIN());
        expect(window.location.assign).toBeCalledWith(
            `${API_HOST}/login?return_to=${window.location.protocol}//${window.location.host}/token`
        );
    });

    it('should fire external url to logout', () => {
        const initialState = undefined;
        window.location.assign = jest.fn();
        authReducer(initialState, actions.LOGOUT());
        expect(window.location.assign).toBeCalledWith(
            `${API_HOST}/logout?return_to=${location.protocol}//${location.host}`
        );
    });
});
