// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { getUserUuid } from "common/getuser";
import { ServiceRepository, createServices, setAuthorizationHeader } from "services/services";
import { setBreadcrumbs } from "store/breadcrumbs/breadcrumbs-actions";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { LinkAccountType, AccountToLink, LinkAccountStatus } from "models/link-account";
import { authActions, getConfig } from "store/auth/auth-action";
import { unionize, ofType, UnionOf } from 'common/unionize';
import { UserResource } from "models/user";
import { GroupResource } from "models/group";
import { LinkAccountPanelError, OriginatingUser } from "./link-account-panel-reducer";
import { login, logout } from "store/auth/auth-action";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { WORKBENCH_LOADING_SCREEN } from 'store/workbench/workbench-actions';

export const linkAccountPanelActions = unionize({
    LINK_INIT: ofType<{
        targetUser: UserResource | undefined
    }>(),
    LINK_LOAD: ofType<{
        originatingUser: OriginatingUser | undefined,
        targetUser: UserResource | undefined,
        targetUserToken: string | undefined,
        userToLink: UserResource | undefined,
        userToLinkToken: string | undefined
    }>(),
    LINK_INVALID: ofType<{
        originatingUser: OriginatingUser | undefined,
        targetUser: UserResource | undefined,
        userToLink: UserResource | undefined,
        error: LinkAccountPanelError
    }>(),
    SET_SELECTED_CLUSTER: ofType<{
        selectedCluster: string
    }>(),
    SET_IS_PROCESSING: ofType<{
        isProcessing: boolean
    }>(),
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

const newServices = (dispatch: Dispatch<any>, token: string) => {
    const config = dispatch<any>(getConfig);
    const svc = createServices(config, { progressFn: () => { }, errorFn: () => { } });
    setAuthorizationHeader(svc, token);
    return svc;
};

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
        dispatch(authActions.INIT_USER({ user, token }));
    };

export const linkFailed = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        // If the link fails, switch to the user account that originated the link operation
        const linkState = getState().linkAccountPanel;
        if (linkState.userToLink && linkState.userToLinkToken && linkState.targetUser && linkState.targetUserToken) {
            if (linkState.originatingUser === OriginatingUser.TARGET_USER) {
                dispatch(switchUser(linkState.targetUser, linkState.targetUserToken));
            }
            else if ((linkState.originatingUser === OriginatingUser.USER_TO_LINK)) {
                dispatch(switchUser(linkState.userToLink, linkState.userToLinkToken));
            }
        }
        services.linkAccountService.removeAccountToLink();
        services.linkAccountService.saveLinkOpStatus(LinkAccountStatus.FAILED);
        location.reload();
    };

export const loadLinkAccountPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            // If there are remote hosts, set the initial selected cluster by getting the first cluster that isn't the local cluster
            if (getState().linkAccountPanel.selectedCluster === undefined) {
                const localCluster = getState().auth.localCluster;
                let selectedCluster = localCluster;
                for (const key in getState().auth.remoteHosts) {
                    if (key !== localCluster) {
                        selectedCluster = key;
                        break;
                    }
                }
                dispatch(linkAccountPanelActions.SET_SELECTED_CLUSTER({ selectedCluster }));
            }

            // First check if an account link operation has completed
            dispatch(checkForLinkStatus());

            // Continue loading the link account panel
            dispatch(setBreadcrumbs([{ label: 'Link account' }]));
            const curUser = getState().auth.user;
            const curToken = getState().auth.apiToken;
            if (curUser && curToken) {

                // If there is link account session data, then the user has logged in a second time
                const linkAccountData = services.linkAccountService.getAccountToLink();
                if (linkAccountData) {

                    dispatch(linkAccountPanelActions.SET_IS_PROCESSING({ isProcessing: true }));
                    const curUserResource = await services.userService.get(curUser.uuid);

                    // Use the token of the user we are getting data for. This avoids any admin/non-admin permissions
                    // issues since a user will always be able to query the api server for their own user data.
                    const svc = newServices(dispatch, linkAccountData.token);
                    const savedUserResource = await svc.userService.get(linkAccountData.userUuid);

                    let params: any;
                    if (linkAccountData.type === LinkAccountType.ACCESS_OTHER_ACCOUNT || linkAccountData.type === LinkAccountType.ACCESS_OTHER_REMOTE_ACCOUNT) {
                        params = {
                            originatingUser: OriginatingUser.USER_TO_LINK,
                            targetUser: curUserResource,
                            targetUserToken: curToken,
                            userToLink: savedUserResource,
                            userToLinkToken: linkAccountData.token
                        };
                    }
                    else if (linkAccountData.type === LinkAccountType.ADD_OTHER_LOGIN || linkAccountData.type === LinkAccountType.ADD_LOCAL_TO_REMOTE) {
                        params = {
                            originatingUser: OriginatingUser.TARGET_USER,
                            targetUser: savedUserResource,
                            targetUserToken: linkAccountData.token,
                            userToLink: curUserResource,
                            userToLinkToken: curToken
                        };
                    }
                    else {
                        throw new Error("Unknown link account type");
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
                            error
                        }));
                        return;
                    }
                }
                else {
                    // If there is no link account session data, set the state to invoke the initial UI
                    const curUserResource = await services.userService.get(curUser.uuid);
                    dispatch(linkAccountPanelActions.LINK_INIT({ targetUser: curUserResource }));
                    return;
                }
            }
        }
        catch (e) {
            dispatch(linkFailed());
        }
        finally {
            dispatch(linkAccountPanelActions.SET_IS_PROCESSING({ isProcessing: false }));
        }
    };

export const startLinking = (t: LinkAccountType) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (!userUuid) { return; }
        const accountToLink = { type: t, userUuid, token: services.authService.getApiToken() } as AccountToLink;
        services.linkAccountService.saveAccountToLink(accountToLink);

        const auth = getState().auth;
        const isLocalUser = auth.user!.uuid.substring(0, 5) === auth.localCluster;
        let homeCluster = auth.localCluster;
        if (isLocalUser && t === LinkAccountType.ACCESS_OTHER_REMOTE_ACCOUNT) {
            homeCluster = getState().linkAccountPanel.selectedCluster!;
        }

        dispatch(logout());
        dispatch(login(auth.localCluster, homeCluster, auth.loginCluster, auth.remoteHosts));
    };

export const getAccountLinkData = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        return services.linkAccountService.getAccountToLink();
    };

export const cancelLinking = (reload: boolean = false) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        let user: UserResource | undefined;
        try {
            // When linking is cancelled switch to the originating user (i.e. the user saved in session data)
            dispatch(progressIndicatorActions.START_WORKING(WORKBENCH_LOADING_SCREEN));
            const linkAccountData = services.linkAccountService.getAccountToLink();
            if (linkAccountData) {
                services.linkAccountService.removeAccountToLink();
                const svc = newServices(dispatch, linkAccountData.token);
                user = await svc.userService.get(linkAccountData.userUuid);
                dispatch(switchUser(user, linkAccountData.token));
                services.linkAccountService.saveLinkOpStatus(LinkAccountStatus.CANCELLED);
            }
        }
        finally {
            if (reload) {
                location.reload();
            }
            else {
                dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
            }
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
                // specified in the request, so change the authorization header accordingly
                const svc = newServices(dispatch, linkState.userToLinkToken);
                await svc.linkAccountService.linkAccounts(linkState.targetUserToken, newGroup.uuid);
                dispatch(switchUser(linkState.targetUser, linkState.targetUserToken));
                services.linkAccountService.removeAccountToLink();
                services.linkAccountService.saveLinkOpStatus(LinkAccountStatus.SUCCESS);
                location.reload();
            }
            catch (e) {
                // If the link operation fails, delete the previously made project
                try {
                    const svc = newServices(dispatch, linkState.targetUserToken);
                    await svc.projectService.delete(newGroup.uuid);
                }
                finally {
                    dispatch(linkFailed());
                }
                throw e;
            }
        }
    };
