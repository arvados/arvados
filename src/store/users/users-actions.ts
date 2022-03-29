// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { ServiceRepository } from "services/services";
import { dialogActions } from 'store/dialog/dialog-actions';
import { startSubmit, reset, stopSubmit } from "redux-form";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { UserResource } from "models/user";
import { getResource } from 'store/resources/resources';
import { navigateTo, navigateToUsers, navigateToRootProject } from "store/navigation/navigation-action";
import { authActions } from 'store/auth/auth-action';
import { getTokenV2 } from "models/api-client-authorization";
import { VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD, VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD } from "store/virtual-machines/virtual-machines-actions";
import { PermissionLevel } from "models/permission";
import { updateResources } from "store/resources/resources-actions";

export const USERS_PANEL_ID = 'usersPanel';
export const USER_ATTRIBUTES_DIALOG = 'userAttributesDialog';
export const USER_CREATE_FORM_NAME = 'userCreateFormName';

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

export const loginAs = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (userUuid === uuid) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: 'You are already logged in as this user',
                kind: SnackbarKind.WARNING
            }));
        } else {
            try {
                const { resources } = getState();
                const data = getResource<UserResource>(uuid)(resources);
                const client = await services.apiClientAuthorizationService.create({ ownerUuid: uuid }, false);
                if (data) {
                    dispatch<any>(authActions.INIT_USER({ user: data, token: getTokenV2(client) }));
                    window.location.reload();
                    dispatch<any>(navigateToRootProject);
                }
            } catch (e) {
                if (e.status === 403) {
                    dispatch(snackbarActions.OPEN_SNACKBAR({
                        message: 'You do not have permission to login as this user',
                        kind: SnackbarKind.WARNING
                    }));
                } else {
                    dispatch(snackbarActions.OPEN_SNACKBAR({
                        message: 'Failed to login as this user',
                        kind: SnackbarKind.ERROR
                    }));
                }
            }
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
