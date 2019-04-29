// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { LinkAccountType, AccountToLink } from "~/models/link-account";
import { logout, saveApiToken, saveUser } from "~/store/auth/auth-action";
import { unionize, ofType, UnionOf } from '~/common/unionize';
import { UserResource } from "~/models/user";
import { navigateToRootProject } from "~/store/navigation/navigation-action";

export const linkAccountPanelActions = unionize({
    LOAD_LINKING: ofType<{ user: UserResource | undefined, userToLink: UserResource | undefined }>(),
    REMOVE_LINKING: {}
});

export type LinkAccountPanelAction = UnionOf<typeof linkAccountPanelActions>;

export const loadLinkAccountPanel = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([{ label: 'Link account'}]));

        const curUser = getState().auth.user;
        if (curUser) {
            services.userService.get(curUser.uuid).then(curUserResource => {
                const linkAccountData = services.linkAccountService.getLinkAccount();
                if (linkAccountData) {
                    services.userService.get(linkAccountData.userUuid).then(savedUserResource => {
                        if (linkAccountData.type === LinkAccountType.ADD_OTHER_LOGIN) {
                            dispatch<any>(linkAccountPanelActions.LOAD_LINKING({ userToLink: curUserResource, user: savedUserResource }));
                        }
                        else if (linkAccountData.type === LinkAccountType.ACCESS_OTHER_ACCOUNT) {
                            dispatch<any>(linkAccountPanelActions.LOAD_LINKING({ userToLink: savedUserResource, user: curUserResource }));
                        }
                        else {
                            throw new Error('Invalid link account type.');
                        }
                    });
                }
                else {
                    dispatch<any>(linkAccountPanelActions.LOAD_LINKING({ userToLink: undefined, user: curUserResource }));
                }
            });
        }
    };

export const saveAccountLinkData = (t: LinkAccountType) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const accountToLink = {type: t, userUuid: services.authService.getUuid(), token: services.authService.getApiToken()} as AccountToLink;
        services.linkAccountService.saveLinkAccount(accountToLink);
        dispatch(logout());
    };

export const getAccountLinkData = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        return services.linkAccountService.getLinkAccount();
    };

export const removeAccountLinkData = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const linkAccountData = services.linkAccountService.getLinkAccount();
        services.linkAccountService.removeLinkAccount();
        dispatch(linkAccountPanelActions.REMOVE_LINKING());
        if (linkAccountData) {
            services.userService.get(linkAccountData.userUuid).then(savedUser => {
                dispatch(setBreadcrumbs([{ label: ''}]));
                dispatch<any>(saveUser(savedUser));
                dispatch<any>(saveApiToken(linkAccountData.token));
                dispatch<any>(navigateToRootProject);
            });
        }
    };

export const linkAccount = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    };