// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, default as unionize, UnionOf } from "unionize";
import { UserDetailsResponse } from "../../services/auth-service/auth-service";

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
