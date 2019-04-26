// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { ApiActions } from "~/services/api/api-actions";
import { AccountToLink } from "~/models/link-account";

export const USER_LINK_ACCOUNT_KEY = 'accountToLink';

export class LinkAccountService {

    constructor(
        protected apiClient: AxiosInstance,
        protected actions: ApiActions) { }

    public saveLinkAccount(account: AccountToLink) {
        sessionStorage.setItem(USER_LINK_ACCOUNT_KEY, JSON.stringify(account));
    }

    public removeLinkAccount() {
        sessionStorage.removeItem(USER_LINK_ACCOUNT_KEY);
    }

    public getLinkAccount() {
        const data = sessionStorage.getItem(USER_LINK_ACCOUNT_KEY);
        return data ? JSON.parse(data) as AccountToLink : undefined;
    }
}