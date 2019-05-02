// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { snackbarActions, SnackbarKind } from "~/store/snackbar/snackbar-actions";
import { LinkAccountType, AccountToLink } from "~/models/link-account";
import { logout, saveApiToken, saveUser } from "~/store/auth/auth-action";
import { unionize, ofType, UnionOf } from '~/common/unionize';
import { UserResource, User } from "~/models/user";
import { navigateToRootProject } from "~/store/navigation/navigation-action";
import { GroupResource } from "~/models/group";
import { LinkAccountPanelError } from "./link-account-panel-reducer";

export const linkAccountPanelActions = unionize({
    INIT: ofType<{ user: UserResource | undefined }>(),
    LOAD: ofType<{
        user: UserResource | undefined,
        userToken: string | undefined,
        userToLink: UserResource | undefined,
        userToLinkToken: string | undefined }>(),
    RESET: {},
    INVALID: ofType<{
        user: UserResource | undefined,
        userToLink: UserResource | undefined,
        error: LinkAccountPanelError }>(),
});

export type LinkAccountPanelAction = UnionOf<typeof linkAccountPanelActions>;

function validateLink(user: UserResource, userToLink: UserResource) {
    if (user.uuid === userToLink.uuid) {
        return LinkAccountPanelError.SAME_USER;
    }
    else if (!user.isAdmin && userToLink.isAdmin) {
        return LinkAccountPanelError.NON_ADMIN;
    }
    return LinkAccountPanelError.NONE;
}

export const loadLinkAccountPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([{ label: 'Link account'}]));

        const curUser = getState().auth.user;
        const curToken = getState().auth.apiToken;
        if (curUser && curToken) {
            const curUserResource = await services.userService.get(curUser.uuid);
            const linkAccountData = services.linkAccountService.getFromSession();

            // If there is link account session data, then the user has logged in a second time
            if (linkAccountData) {
                dispatch<any>(saveApiToken(linkAccountData.token));
                const savedUserResource = await services.userService.get(linkAccountData.userUuid);
                dispatch<any>(saveApiToken(curToken));

                let params: any;
                if (linkAccountData.type === LinkAccountType.ACCESS_OTHER_ACCOUNT) {
                    params = {
                        user: savedUserResource,
                        userToken: linkAccountData.token,
                        userToLink: curUserResource,
                        userToLinkToken: curToken
                    };
                }
                else if (linkAccountData.type === LinkAccountType.ADD_OTHER_LOGIN) {
                    params = {
                        user: curUserResource,
                        userToken: curToken,
                        userToLink: savedUserResource,
                        userToLinkToken: linkAccountData.token
                    };
                }
                else {
                    throw new Error("Invalid link account type.");
                }

                const error = validateLink(params.user, params.userToLink);
                if (error === LinkAccountPanelError.NONE) {
                    dispatch<any>(linkAccountPanelActions.LOAD(params));
                }
                else {
                    dispatch<any>(linkAccountPanelActions.INVALID({
                        user: params.user,
                        userToLink: params.userToLink,
                        error}));
                    return;
                }
            }
            else {
                // If there is no link account session data, set the state to invoke the initial UI
                dispatch<any>(linkAccountPanelActions.INIT({
                    user: curUserResource }
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
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            const linkAccountData = services.linkAccountService.getFromSession();
            if (linkAccountData) {
                const savedUser = await services.userService.get(linkAccountData.userUuid);
                dispatch<any>(saveUser(savedUser));
                dispatch<any>(saveApiToken(linkAccountData.token));
            }
        }
        finally {
            dispatch<any>(navigateToRootProject);
            dispatch(linkAccountPanelActions.RESET());
            services.linkAccountService.removeFromSession();
        }
    };

export const linkAccount = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const linkState = getState().linkAccountPanel;
        const currentToken = getState().auth.apiToken;
        if (linkState.userToLink && linkState.userToLinkToken && linkState.user && linkState.userToken && currentToken) {

            // First create a project owned by the "userToLink" to accept everything from the current user
            const projectName = `Migrated from ${linkState.user.email} (${linkState.user.uuid})`;
            let newGroup: GroupResource;
            try {
                dispatch<any>(saveApiToken(linkState.userToLinkToken));
                newGroup = await services.projectService.create({
                    name: projectName,
                    ensure_unique_name: true
                });
            }
            catch (e) {
                dispatch<any>(saveApiToken(currentToken));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Account link failed.', kind: SnackbarKind.ERROR , hideDuration: 3000
                }));
                throw e;
            }

            try {
                // Use the token of the account that is getting merged to call the merge api
                dispatch<any>(saveApiToken(linkState.userToken));
                await services.linkAccountService.linkAccounts(linkState.userToLinkToken, newGroup.uuid);

                // If the link was successful, switch to the account that was merged with
                dispatch<any>(saveUser(linkState.userToLink));
                dispatch<any>(saveApiToken(linkState.userToLinkToken));
            }
            catch(e) {
                // If the link operation fails, delete the previously made project
                // and stay logged in to the current account.
                try {
                    dispatch<any>(saveApiToken(linkState.userToLinkToken));
                    await services.projectService.delete(newGroup.uuid);
                }
                finally {
                    dispatch<any>(saveApiToken(currentToken));
                    dispatch(snackbarActions.OPEN_SNACKBAR({
                        message: 'Account link failed.', kind: SnackbarKind.ERROR , hideDuration: 3000
                    }));
                }
                throw e;
            }
            finally {
                dispatch<any>(navigateToRootProject);
                services.linkAccountService.removeFromSession();
                dispatch(linkAccountPanelActions.RESET());
            }
        }
    };