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
    LOAD_LINKING: ofType<{
        user: UserResource | undefined,
        userToken: string | undefined,
        userToLink: UserResource | undefined,
        userToLinkToken: string | undefined }>(),
    RESET_LINKING: {}
});

export type LinkAccountPanelAction = UnionOf<typeof linkAccountPanelActions>;

export const loadLinkAccountPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([{ label: 'Link account'}]));

        const curUser = getState().auth.user;
        const curToken = getState().auth.apiToken;
        if (curUser && curToken) {
            const curUserResource = await services.userService.get(curUser.uuid);
            const linkAccountData = services.linkAccountService.getFromSession();

            // If there is link account data, then the user has logged in a second time
            if (linkAccountData) {
                // Use the saved token to make the api call to override the current users permissions
                dispatch<any>(saveApiToken(linkAccountData.token));
                const savedUserResource = await services.userService.get(linkAccountData.userUuid);
                dispatch<any>(saveApiToken(curToken));
                if (linkAccountData.type === LinkAccountType.ACCESS_OTHER_ACCOUNT) {
                    const params = {
                        user: savedUserResource,
                        userToken: linkAccountData.token,
                        userToLink: curUserResource,
                        userToLinkToken: curToken
                    };
                    dispatch<any>(linkAccountPanelActions.LOAD_LINKING(params));
                }
                else if (linkAccountData.type === LinkAccountType.ADD_OTHER_LOGIN) {
                    const params = {
                        user: curUserResource,
                        userToken: curToken,
                        userToLink: savedUserResource,
                        userToLinkToken: linkAccountData.token
                    };
                    dispatch<any>(linkAccountPanelActions.LOAD_LINKING(params));
                }
                else {
                    throw new Error("Invalid link account type.");
                }
            }
            else {
                // If there is no link account session data, set the state to invoke the initial UI
                dispatch<any>(linkAccountPanelActions.LOAD_LINKING({
                    user: curUserResource,
                    userToken: curToken,
                    userToLink: undefined,
                    userToLinkToken: undefined }
                ));
            }
        }
    };

export const saveAccountLinkData = (t: LinkAccountType) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const accountToLink = {type: t, userUuid: services.authService.getUuid(), token: services.authService.getApiToken()} as AccountToLink;
        services.linkAccountService.saveToSession(accountToLink);
        dispatch(logout());
    };

export const getAccountLinkData = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        return services.linkAccountService.getFromSession();
    };

export const removeAccountLinkData = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const linkAccountData = services.linkAccountService.getFromSession();
        services.linkAccountService.removeFromSession();
        dispatch(linkAccountPanelActions.RESET_LINKING());
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
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const linkState = getState().linkAccountPanel;
        const currentToken = getState().auth.apiToken;
        if (linkState.userToLink && linkState.userToLinkToken && linkState.user && linkState.userToken && currentToken) {

            // First create a project owned by the "userToLink" to accept everything from the current user
            const projectName = `Migrated from ${linkState.user.email} (${linkState.user.uuid})`;
            dispatch<any>(saveApiToken(linkState.userToLinkToken));
            const newGroup = await services.projectService.create({
                name: projectName,
                ensure_unique_name: true
            });
            dispatch<any>(saveApiToken(currentToken));

            try {
                dispatch<any>(saveApiToken(linkState.userToken));
                await services.linkAccountService.linkAccounts(linkState.userToLinkToken, newGroup.uuid);
                dispatch<any>(saveApiToken(currentToken));

                // If the link was successful, switch to the account that was merged with
                if (linkState.userToLink && linkState.userToLinkToken) {
                    dispatch<any>(saveUser(linkState.userToLink));
                    dispatch<any>(saveApiToken(linkState.userToLinkToken));
                    dispatch<any>(navigateToRootProject);
                }
                services.linkAccountService.removeFromSession();
                dispatch(linkAccountPanelActions.RESET_LINKING());
            }
            catch(e) {
                // If the account link operation fails, delete the previously made project
                // and reset the link account state. The user will have to restart the process.
                dispatch<any>(saveApiToken(linkState.userToLinkToken));
                services.projectService.delete(newGroup.uuid);
                dispatch<any>(saveApiToken(currentToken));
                services.linkAccountService.removeFromSession();
                dispatch(linkAccountPanelActions.RESET_LINKING());
                throw e;
            }
        }
    };