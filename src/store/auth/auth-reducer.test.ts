// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authReducer, AuthState } from "./auth-reducer";
import { AuthAction, authActions } from "./auth-action";
import {
    API_TOKEN_KEY,
    USER_EMAIL_KEY,
    USER_FIRST_NAME_KEY,
    USER_LAST_NAME_KEY,
    USER_OWNER_UUID_KEY,
    USER_UUID_KEY
} from "../../services/auth-service/auth-service";

import 'jest-localstorage-mock';
import { createServices } from "../../services/services";

describe('auth-reducer', () => {
    let reducer: (state: AuthState | undefined, action: AuthAction) => any;

    beforeAll(() => {
        localStorage.clear();
        reducer = authReducer(createServices("/arvados/v1"));
    });

    it('should return default state on initialisation', () => {
        const initialState = undefined;
        const state = reducer(initialState, authActions.INIT());
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
        localStorage.setItem(USER_UUID_KEY, "uuid");
        localStorage.setItem(USER_OWNER_UUID_KEY, "ownerUuid");

        const state = reducer(initialState, authActions.INIT());
        expect(state).toEqual({
            apiToken: "token",
            user: {
                email: "test@test.com",
                firstName: "John",
                lastName: "Doe",
                uuid: "uuid",
                ownerUuid: "ownerUuid"
            }
        });
    });

    it('should store token in local storage', () => {
        const initialState = undefined;

        const state = reducer(initialState, authActions.SAVE_API_TOKEN("token"));
        expect(state).toEqual({
            apiToken: "token",
            user: undefined
        });

        expect(localStorage.getItem(API_TOKEN_KEY)).toBe("token");
    });

    it('should set user details on success fetch', () => {
        const initialState = undefined;

        const user = {
            email: "test@test.com",
            firstName: "John",
            lastName: "Doe",
            uuid: "uuid",
            ownerUuid: "ownerUuid"
        };

        const state = reducer(initialState, authActions.USER_DETAILS_SUCCESS(user));
        expect(state).toEqual({
            apiToken: undefined,
            user: {
                email: "test@test.com",
                firstName: "John",
                lastName: "Doe",
                uuid: "uuid",
                ownerUuid: "ownerUuid",
            }
        });

        expect(localStorage.getItem(API_TOKEN_KEY)).toBe("token");
    });

    it('should fire external url to login', () => {
        const initialState = undefined;
        window.location.assign = jest.fn();
        reducer(initialState, authActions.LOGIN());
        expect(window.location.assign).toBeCalledWith(
            `/login?return_to=${window.location.protocol}//${window.location.host}/token`
        );
    });

    it('should fire external url to logout', () => {
        const initialState = undefined;
        window.location.assign = jest.fn();
        reducer(initialState, authActions.LOGOUT());
        expect(window.location.assign).toBeCalledWith(
            `/logout?return_to=${location.protocol}//${location.host}`
        );
    });
});
