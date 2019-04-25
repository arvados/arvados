// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { LinkAccountType, AccountToLink } from "~/models/link-account";
import { logout } from "~/store/auth/auth-action";

export const loadLinkAccountPanel = () =>
    (dispatch: Dispatch<any>) => {
       dispatch(setBreadcrumbs([{ label: 'Link account'}]));
    };

export const saveAccountLinkData = (t: LinkAccountType) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const accountToLink = {type: t, userToken: services.authService.getApiToken()} as AccountToLink;
        sessionStorage.setItem("accountToLink", JSON.stringify(accountToLink));
        dispatch(logout());
    };