// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { serverApi } from "../common/server-api";
import { ofType, default as unionize, UnionOf } from "unionize";
import { Dispatch } from "redux";

export interface UserDetailsResponse {
    email: string;
    first_name: string;
    last_name: string;
    is_admin: boolean;
}

const actions = unionize({
    SAVE_API_TOKEN: ofType<string>(),
    LOGIN: {},
    LOGOUT: {},
    INIT: {},
    USER_DETAILS_REQUEST: {},
    USER_DETAILS_SUCCESS: ofType<UserDetailsResponse>()
}, {
    tag: 'type',
    value: 'payload'
});

export type AuthAction = UnionOf<typeof actions>;
export default actions;

export const getUserDetails = () => (dispatch: Dispatch) => {
    dispatch(actions.USER_DETAILS_REQUEST());
    serverApi
        .get<UserDetailsResponse>('/users/current')
        .then(resp => {
            dispatch(actions.USER_DETAILS_SUCCESS(resp.data));
        })
        // .catch(err => {
        // });
};


