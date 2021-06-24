// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { ApiActions } from "services/api/api-actions";
import { AccountToLink, LinkAccountStatus } from "models/link-account";
import { CommonService } from "services/common-service/common-service";

export const USER_LINK_ACCOUNT_KEY = 'accountToLink';
export const ACCOUNT_LINK_STATUS_KEY = 'accountLinkStatus';

export class LinkAccountService {

    constructor(
        protected serverApi: AxiosInstance,
        protected actions: ApiActions) { }

    public saveAccountToLink(account: AccountToLink) {
        sessionStorage.setItem(USER_LINK_ACCOUNT_KEY, JSON.stringify(account));
    }

    public removeAccountToLink() {
        sessionStorage.removeItem(USER_LINK_ACCOUNT_KEY);
    }

    public getAccountToLink() {
        const data = sessionStorage.getItem(USER_LINK_ACCOUNT_KEY);
        return data ? JSON.parse(data) as AccountToLink : undefined;
    }

    public saveLinkOpStatus(status: LinkAccountStatus) {
        sessionStorage.setItem(ACCOUNT_LINK_STATUS_KEY, JSON.stringify(status));
    }

    public removeLinkOpStatus() {
        sessionStorage.removeItem(ACCOUNT_LINK_STATUS_KEY);
    }

    public getLinkOpStatus() {
        const data = sessionStorage.getItem(ACCOUNT_LINK_STATUS_KEY);
        return data ? JSON.parse(data) as LinkAccountStatus : undefined;
    }

    public linkAccounts(newUserToken: string, newGroupUuid: string) {
        const params = {
            new_user_token: newUserToken,
            new_owner_uuid: newGroupUuid,
            redirect_to_new_user: true
        };
        return CommonService.defaultResponse(
            this.serverApi.post('/users/merge', params),
            this.actions,
            false
        );
    }
}
