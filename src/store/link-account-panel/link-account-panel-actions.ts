// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { snackbarActions, SnackbarKind } from "~/store/snackbar/snackbar-actions";
import { LinkAccountType, AccountToLink, LinkAccountStatus } from "~/models/link-account";
import { saveApiToken, saveUser } from "~/store/auth/auth-action";
import { unionize, ofType, UnionOf } from '~/common/unionize';
import { UserResource } from "~/models/user";
import { GroupResource } from "~/models/group";
import { LinkAccountPanelError, OriginatingUser } from "./link-account-panel-reducer";
import { login, logout } from "~/store/auth/auth-action";
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { WORKBENCH_LOADING_SCREEN } from '~/store/workbench/workbench-actions';

export const linkAccountPanelActions = unionize({
    LINK_INIT: ofType<{ targetUser: UserResource | undefined }>(),
    LINK_LOAD: ofType<{
        originatingUser: OriginatingUser | undefined,
        targetUser: UserResource | undefined,
        targetUserToken: string | undefined,
        userToLink: UserResource | undefined,
        userToLinkToken: string | undefined }>(),
    LINK_INVALID: ofType<{
        originatingUser: OriginatingUser | undefined,
        targetUser: UserResource | undefined,
        userToLink: UserResource | undefined,
        error: LinkAccountPanelError }>(),
    HAS_SESSION_DATA: {}
});

export type LinkAccountPanelAction = UnionOf<typeof linkAccountPanelActions>;

function validateLink(userToLink: UserResource, targetUser: UserResource) {
    if (userToLink.uuid === targetUser.uuid) {
        return LinkAccountPanelError.SAME_USER;
    }
    else if (userToLink.isAdmin && !targetUser.isAdmin) {
        return LinkAccountPanelError.NON_ADMIN;
    }
    else if (!targetUser.isActive) {
        return LinkAccountPanelError.INACTIVE;
    }
    return LinkAccountPanelError.NONE;
}

export const checkForLinkStatus = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const status = services.linkAccountService.getLinkOpStatus();
        if (status !== undefined) {
            let msg: string;
            let msgKind: SnackbarKind;
            if (status.valueOf() === LinkAccountStatus.CANCELLED) {
                msg = "Account link cancelled!", msgKind = SnackbarKind.INFO;
            }
            else if (status.valueOf() === LinkAccountStatus.FAILED) {
                msg = "Account link failed!", msgKind = SnackbarKind.ERROR;
            }
            else if (status.valueOf() === LinkAccountStatus.SUCCESS) {
                msg = "Account link success!", msgKind = SnackbarKind.SUCCESS;
            }
            else {
                msg = "Unknown Error!", msgKind = SnackbarKind.ERROR;
            }
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: msg, kind: msgKind, hideDuration: 3000 }));
            services.linkAccountService.removeLinkOpStatus();
        }
    };

export const switchUser = (user: UserResource, token: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(saveUser(user));
        dispatch(saveApiToken(token));
    };

export const linkFailed = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        // If the link fails, switch to the user account that originated the link operation
        const linkState = getState().linkAccountPanel;
        if (linkState.userToLink && linkState.userToLinkToken && linkState.targetUser && linkState.targetUserToken) {
            if (linkState.originatingUser === OriginatingUser.TARGET_USER) {
                dispatch(switchUser(linkState.targetUser, linkState.targetUserToken));
                dispatch(linkAccountPanelActions.LINK_INIT({targetUser: linkState.targetUser}));
            }
            else if ((linkState.originatingUser === OriginatingUser.USER_TO_LINK)) {
                dispatch(switchUser(linkState.userToLink, linkState.userToLinkToken));
                dispatch(linkAccountPanelActions.LINK_INIT({targetUser: linkState.userToLink}));
            }
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Account link failed.', kind: SnackbarKind.ERROR , hideDuration: 3000 }));
        }
        services.linkAccountService.removeAccountToLink();
    };

export const loadLinkAccountPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        // First check if an account link operation has completed
        dispatch(checkForLinkStatus());

        // Continue loading the link account panel
        dispatch(setBreadcrumbs([{ label: 'Link account'}]));
        const curUser = getState().auth.user;
        const curToken = getState().auth.apiToken;
        if (curUser && curToken) {
            const curUserResource = await services.userService.get(curUser.uuid);
            const linkAccountData = services.linkAccountService.getAccountToLink();

            // If there is link account session data, then the user has logged in a second time
            if (linkAccountData) {

                // Use the token of the user we are getting data for. This avoids any admin/non-admin permissions
                // issues since a user will always be able to query the api server for their own user data.
                dispatch(saveApiToken(linkAccountData.token));
                const savedUserResource = await services.userService.get(linkAccountData.userUuid);
                dispatch(saveApiToken(curToken));

                let params: any;
                if (linkAccountData.type === LinkAccountType.ACCESS_OTHER_ACCOUNT) {
                    params = {
                        originatingUser: OriginatingUser.USER_TO_LINK,
                        targetUser: curUserResource,
                        targetUserToken: curToken,
                        userToLink: savedUserResource,
                        userToLinkToken: linkAccountData.token
                    };
                }
                else if (linkAccountData.type === LinkAccountType.ADD_OTHER_LOGIN) {
                    params = {
                        originatingUser: OriginatingUser.TARGET_USER,
                        targetUser: savedUserResource,
                        targetUserToken: linkAccountData.token,
                        userToLink: curUserResource,
                        userToLinkToken: curToken
                    };
                }
                else {
                    // This should never really happen, but just in case, switch to the user that
                    // originated the linking operation (i.e. the user saved in session data)
                    dispatch(switchUser(savedUserResource, linkAccountData.token));
                    services.linkAccountService.removeAccountToLink();
                    dispatch(linkAccountPanelActions.LINK_INIT({targetUser:savedUserResource}));
                }

                dispatch(switchUser(params.targetUser, params.targetUserToken));
                const error = validateLink(params.userToLink, params.targetUser);
                if (error === LinkAccountPanelError.NONE) {
                    dispatch(linkAccountPanelActions.LINK_LOAD(params));
                }
                else {
                    dispatch(linkAccountPanelActions.LINK_INVALID({
                        originatingUser: params.originatingUser,
                        targetUser: params.targetUser,
                        userToLink: params.userToLink,
                        error}));
                    return;
                }
            }
            else {
                // If there is no link account session data, set the state to invoke the initial UI
                dispatch(linkAccountPanelActions.LINK_INIT({ targetUser: curUserResource }));
                return;
            }
        }
    };

export const startLinking = (t: LinkAccountType) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const accountToLink = {type: t, userUuid: services.authService.getUuid(), token: services.authService.getApiToken()} as AccountToLink;
        services.linkAccountService.saveAccountToLink(accountToLink);
        const auth = getState().auth;
        dispatch(logout());
        dispatch(login(auth.localCluster, auth.homeCluster, auth.remoteHosts));
    };

export const getAccountLinkData = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        return services.linkAccountService.getAccountToLink();
    };

export const cancelLinking = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        let user: UserResource | undefined;
        try {
            dispatch(progressIndicatorActions.START_WORKING(WORKBENCH_LOADING_SCREEN));
            // When linking is cancelled switch to the originating user (i.e. the user saved in session data)
            const linkAccountData = services.linkAccountService.getAccountToLink();
            if (linkAccountData) {
                dispatch(saveApiToken(linkAccountData.token));
                user = await services.userService.get(linkAccountData.userUuid);
                dispatch(switchUser(user, linkAccountData.token));
            }
        }
        finally {
            services.linkAccountService.removeAccountToLink();
            dispatch(linkAccountPanelActions.LINK_INIT({targetUser:user}));
            dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
        }
    };

export const linkAccount = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const linkState = getState().linkAccountPanel;
        if (linkState.userToLink && linkState.userToLinkToken && linkState.targetUser && linkState.targetUserToken) {

            // First create a project owned by the target user
            const projectName = `Migrated from ${linkState.userToLink.email} (${linkState.userToLink.uuid})`;
            let newGroup: GroupResource;
            try {
                newGroup = await services.projectService.create({
                    name: projectName,
                    ensure_unique_name: true
                });
            }
            catch (e) {
                dispatch(linkFailed());
                throw e;
            }

            try {
                // The merge api links the user sending the request into the user
                // specified in the request, so switch users for this api call
                dispatch(saveApiToken(linkState.userToLinkToken));
                await services.linkAccountService.linkAccounts(linkState.targetUserToken, newGroup.uuid);
                dispatch(switchUser(linkState.targetUser, linkState.targetUserToken));
                services.linkAccountService.removeAccountToLink();
                services.linkAccountService.saveLinkOpStatus(LinkAccountStatus.SUCCESS);
                location.reload();
            }
            catch(e) {
                // If the link operation fails, delete the previously made project
                try {
                    dispatch(saveApiToken(linkState.targetUserToken));
                    await services.projectService.delete(newGroup.uuid);
                }
                finally {
                    dispatch(linkFailed());
                }
                throw e;
            }
        }
    };