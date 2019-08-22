// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { RootState } from '~/store/store';
import { ServiceRepository } from "~/services/services";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { startSubmit, reset } from "redux-form";
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { UserResource } from "~/models/user";
import { getResource } from '~/store/resources/resources';
import { navigateTo, navigateToUsers, navigateToRootProject } from "~/store/navigation/navigation-action";
import { saveApiToken } from '~/store/auth/auth-action';

export const USERS_PANEL_ID = 'usersPanel';
export const USER_ATTRIBUTES_DIALOG = 'userAttributesDialog';
export const USER_CREATE_FORM_NAME = 'userCreateFormName';
export const USER_MANAGEMENT_DIALOG = 'userManageDialog';
export const SETUP_SHELL_ACCOUNT_DIALOG = 'setupShellAccountDialog';

export interface UserCreateFormDialogData {
    email: string;
    virtualMachineName: string;
    groupVirtualMachine: string;
}

export const openUserAttributes = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: USER_ATTRIBUTES_DIALOG, data }));
    };

export const openUserManagement = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: USER_MANAGEMENT_DIALOG, data }));
    };

export const openSetupShellAccount = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const user = getResource<UserResource>(uuid)(resources);
        const virtualMachines = await services.virtualMachineService.list();
        dispatch(dialogActions.CLOSE_DIALOG({ id: USER_MANAGEMENT_DIALOG }));
        dispatch(dialogActions.OPEN_DIALOG({ id: SETUP_SHELL_ACCOUNT_DIALOG, data: { user, ...virtualMachines } }));
    };

export const loginAs = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        if (data) {
            services.authService.saveUser(data);
        }
        const client = await services.apiClientAuthorizationService.create({ ownerUuid: uuid });
        dispatch<any>(saveApiToken(`v2/${client.uuid}/${client.apiToken}`));
        location.reload();
        dispatch<any>(navigateToRootProject);
    };

export const openUserCreateDialog = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = await services.authService.getUuid();
        const user = await services.userService.get(userUuid!);
        const virtualMachines = await services.virtualMachineService.list();
        dispatch(reset(USER_CREATE_FORM_NAME));
        dispatch(dialogActions.OPEN_DIALOG({ id: USER_CREATE_FORM_NAME, data: { user, ...virtualMachines } }));
    };

export const openUserProjects = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(navigateTo(uuid));
    };


export const createUser = (user: UserCreateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(USER_CREATE_FORM_NAME));
        try {
            const newUser = await services.userService.create({ ...user });
            dispatch(dialogActions.CLOSE_DIALOG({ id: USER_CREATE_FORM_NAME }));
            dispatch(reset(USER_CREATE_FORM_NAME));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "User has been successfully created.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            dispatch<any>(loadUsersPanel());
            dispatch(userBindedActions.REQUEST_ITEMS());
            return newUser;
        } catch (e) {
            return;
        }
    };

export const openUserPanel = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user;
        if (user && user.isAdmin) {
            dispatch<any>(navigateToUsers);
        } else {
            dispatch<any>(navigateToRootProject);
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000 }));
        }
    };

export const toggleIsActive = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        const isActive = data!.isActive;
        const newActivity = await services.userService.update(uuid, { isActive: !isActive });
        dispatch<any>(loadUsersPanel());
        return newActivity;
    };

export const toggleIsAdmin = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        const isAdmin = data!.isAdmin;
        const newActivity = await services.userService.update(uuid, { isAdmin: !isAdmin });
        dispatch<any>(loadUsersPanel());
        return newActivity;
    };

export const userBindedActions = bindDataExplorerActions(USERS_PANEL_ID);

export const loadUsersData = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        await services.userService.list();
    };

export const loadUsersPanel = () =>
    (dispatch: Dispatch) => {
        dispatch(userBindedActions.REQUEST_ITEMS());
    };
