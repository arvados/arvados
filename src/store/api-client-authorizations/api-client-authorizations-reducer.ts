// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { 
    apiClientAuthorizationsActions, 
    ApiClientAuthorizationsActions 
} from '~/store/api-client-authorizations/api-client-authorizations-actions';
import { ApiClientAuthorization } from '~/models/api-client-authorization';

export type ApiClientAuthorizationsState = ApiClientAuthorization[];

const initialState: ApiClientAuthorizationsState = [];

export const apiClientAuthorizationsReducer = 
    (state: ApiClientAuthorizationsState = initialState, action: ApiClientAuthorizationsActions): ApiClientAuthorizationsState =>
        apiClientAuthorizationsActions.match(action, {
            SET_API_CLIENT_AUTHORIZATIONS: apiClientAuthorizations => apiClientAuthorizations,
            REMOVE_API_CLIENT_AUTHORIZATION: (uuid: string) => 
                state.filter((apiClientAuthorization) => apiClientAuthorization.uuid !== uuid),
            default: () => state
        });