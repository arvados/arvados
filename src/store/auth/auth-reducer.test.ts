// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { authReducer, AuthState } from "./auth-reducer";
import { AuthAction, authActions } from "./auth-action";

import 'jest-localstorage-mock';
import { createServices } from "~/services/services";

describe('auth-reducer', () => {
    let reducer: (state: AuthState | undefined, action: AuthAction) => any;

    beforeAll(() => {
        localStorage.clear();
        reducer = authReducer(createServices("/arvados/v1"));
    });

    it('should correctly initialise state', () => {
        const initialState = undefined;
        const user = {
            email: "test@test.com",
            firstName: "John",
            lastName: "Doe",
            uuid: "uuid",
            ownerUuid: "ownerUuid"
        };
        const state = reducer(initialState, authActions.INIT({user, token: "token"}));
        expect(state).toEqual({
            apiToken: "token",
            user
        });
    });

    it('should save api token', () => {
        const initialState = undefined;

        const state = reducer(initialState, authActions.SAVE_API_TOKEN("token"));
        expect(state).toEqual({
            apiToken: "token",
            user: undefined
        });
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
    });
});
