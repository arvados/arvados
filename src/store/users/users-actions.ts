// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { ServiceRepository } from "services/services";
import { dialogActions } from 'store/dialog/dialog-actions';
import { startSubmit, reset, initialize, stopSubmit } from "redux-form";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { UserResource } from "models/user";
import { getResource } from 'store/resources/resources';
import { navigateTo, navigateToUsers, navigateToRootProject } from "store/navigation/navigation-action";
import { authActions } from 'store/auth/auth-action';
import { getTokenV2 } from "models/api-client-authorization";
import { AddLoginFormData, VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD, VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD, VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD } from "store/virtual-machines/virtual-machines-actions";
import { PermissionLevel } from "models/permission";
import { updateResources } from "store/resources/resources-actions";

export const USERS_PANEL_ID = 'usersPanel';
export const USER_ATTRIBUTES_DIALOG = 'userAttributesDialog';
export const USER_CREATE_FORM_NAME = 'userCreateFormName';
export const USER_MANAGEMENT_DIALOG = 'userManageDialog';
export const SETUP_SHELL_ACCOUNT_DIALOG = 'setupShellAccountDialog';

export interface UserCreateFormDialogData {
    email: string;
    [VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD]: string;
    [VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD]: string[];
}

export const userBindedActions = bindDataExplorerActions(USERS_PANEL_ID);

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
        dispatch(initialize(SETUP_SHELL_ACCOUNT_DIALOG, {[VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD]: user, [VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD]: []}));
        dispatch(dialogActions.OPEN_DIALOG({ id: SETUP_SHELL_ACCOUNT_DIALOG, data: virtualMachines }));
    };

export const loginAs = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        const client = await services.apiClientAuthorizationService.create({ ownerUuid: uuid });
        if (data) {
            dispatch<any>(authActions.INIT_USER({ user: data, token: getTokenV2(client) }));
            window.location.reload();
            dispatch<any>(navigateToRootProject);
        }
    };

export const openUserCreateDialog = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (!userUuid) { return; }
        const user = await services.userService.get(userUuid!);
        const virtualMachines = await services.virtualMachineService.list();
        dispatch(reset(USER_CREATE_FORM_NAME));
        dispatch(dialogActions.OPEN_DIALOG({ id: USER_CREATE_FORM_NAME, data: { user, ...virtualMachines } }));
    };

export const openUserProjects = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(navigateTo(uuid));
    };

export const createUser = (data: UserCreateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(USER_CREATE_FORM_NAME));
        try {
            const newUser = await services.userService.create({
                email: data.email,
            });
            dispatch(updateResources([newUser]));

            if (data[VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD]) {
                const permission = await services.permissionService.create({
                    headUuid: data[VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD],
                    tailUuid: newUser.uuid,
                    name: PermissionLevel.CAN_LOGIN,
                    properties: {
                        username: newUser.username,
                        groups: data.groups,
                    }
                });
                dispatch(updateResources([permission]));
            }

            dispatch(dialogActions.CLOSE_DIALOG({ id: USER_CREATE_FORM_NAME }));
            dispatch(reset(USER_CREATE_FORM_NAME));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "User has been successfully created.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            dispatch<any>(loadUsersPanel());
            dispatch(userBindedActions.REQUEST_ITEMS());
            return newUser;
        } catch (e) {
            return;
        } finally {
            dispatch(stopSubmit(USER_CREATE_FORM_NAME));
        }
    };

export const setupUserVM = (setupData: AddLoginFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(SETUP_SHELL_ACCOUNT_DIALOG));
        try {
            const userResource = await services.userService.get(setupData.user.uuid);

            const resources = await services.userService.setup(setupData.user.uuid);
            dispatch(updateResources(resources.items));

            const permission = await services.permissionService.create({
                headUuid: setupData.vmUuid,
                tailUuid: userResource.uuid,
                name: PermissionLevel.CAN_LOGIN,
                properties: {
                    username: userResource.username,
                    groups: setupData.groups,
                }
            });
            dispatch(updateResources([permission]));

            dispatch(dialogActions.CLOSE_DIALOG({ id: SETUP_SHELL_ACCOUNT_DIALOG }));
            dispatch(reset(SETUP_SHELL_ACCOUNT_DIALOG));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "User has been added to VM.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            dispatch(stopSubmit(SETUP_SHELL_ACCOUNT_DIALOG));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
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
        let newActivity;
        if (isActive) {
            newActivity = await services.userService.unsetup(uuid);
        } else {
            newActivity = await services.userService.update(uuid, { isActive: true });
        }
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

export const loadUsersPanel = () =>
    (dispatch: Dispatch) => {
        dispatch(userBindedActions.REQUEST_ITEMS());
    };
