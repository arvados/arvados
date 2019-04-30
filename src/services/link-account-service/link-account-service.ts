// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { ApiActions } from "~/services/api/api-actions";
import { AccountToLink } from "~/models/link-account";
import { CommonService } from "~/services/common-service/common-service";
import { AuthService } from "../auth-service/auth-service";

export const USER_LINK_ACCOUNT_KEY = 'accountToLink';

export class LinkAccountService {

    constructor(
        protected serverApi: AxiosInstance,
        protected actions: ApiActions) { }

    public saveToSession(account: AccountToLink) {
        sessionStorage.setItem(USER_LINK_ACCOUNT_KEY, JSON.stringify(account));
    }

    public removeFromSession() {
        sessionStorage.removeItem(USER_LINK_ACCOUNT_KEY);
    }

    public getFromSession() {
        const data = sessionStorage.getItem(USER_LINK_ACCOUNT_KEY);
        return data ? JSON.parse(data) as AccountToLink : undefined;
    }

    public linkAccounts(newUserToken: string, newGroupUuid: string) {
        const params = {
            new_user_token: newUserToken,
            new_owner_uuid: newGroupUuid,
            redirect_to_new_user: true
        };
        return CommonService.defaultResponse(
            this.serverApi.post('/users/merge/', params),
            this.actions,
            false
        );
    }
}