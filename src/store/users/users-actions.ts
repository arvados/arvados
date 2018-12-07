// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { RootState } from '~/store/store';
import { ServiceRepository } from "~/services/services";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { startSubmit, reset, stopSubmit } from "redux-form";
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/services/common-service/common-resource-service";
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { UserResource } from "~/models/user";
import { getResource } from '~/store/resources/resources';
import { navigateToProject } from "~/store/navigation/navigation-action";

export const USERS_PANEL_ID = 'usersPanel';
export const USER_ATTRIBUTES_DIALOG = 'userAttributesDialog';
export const USER_CREATE_FORM_NAME = 'repositoryCreateFormName';

export interface UserCreateFormDialogData {
    email: string;
    identityUrl: string;
    virtualMachineName: string;
    groupVirtualMachine: string;
}

export const openUserAttributes = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: USER_ATTRIBUTES_DIALOG, data }));
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
        dispatch<any>(navigateToProject(uuid));
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
            return ;
        }
    };

export const toggleIsActive = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        const isActive = data!.isActive;
        const newActivity = await services.userService.update(uuid, { ...data, isActive: !isActive });
        dispatch<any>(loadUsersPanel());
        return newActivity;
    };

export const toggleIsAdmin = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        const isAdmin = data!.isAdmin;
        const newActivity = await services.userService.update(uuid, { ...data, isAdmin: !isAdmin });
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