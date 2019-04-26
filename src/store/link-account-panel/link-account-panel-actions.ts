// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { LinkAccountType, AccountToLink } from "~/models/link-account";
import { logout } from "~/store/auth/auth-action";
import { unionize, ofType, UnionOf } from '~/common/unionize';

export const linkAccountPanelActions = unionize({
    LOAD_LINKING: ofType<AccountToLink>(),
    REMOVE_LINKING: {}
});

export type LinkAccountPanelAction = UnionOf<typeof linkAccountPanelActions>;

export const loadLinkAccountPanel = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([{ label: 'Link account'}]));

        const linkAccountData = services.linkAccountService.getLinkAccount();
        if (linkAccountData) {
            dispatch(linkAccountPanelActions.LOAD_LINKING(linkAccountData));
        }
    };

export const saveAccountLinkData = (t: LinkAccountType) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const accountToLink = {type: t, userToken: services.authService.getApiToken()} as AccountToLink;
        services.linkAccountService.saveLinkAccount(accountToLink);
        dispatch(logout());
    };

export const getAccountLinkData = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        return services.linkAccountService.getLinkAccount();
    };

export const removeAccountLinkData = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        services.linkAccountService.removeLinkAccount();
        dispatch(linkAccountPanelActions.REMOVE_LINKING());
    };