// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from 'store/store';
import { ServiceRepository } from "services/services";
import { navigateToUserVirtualMachines, navigateToAdminVirtualMachines, navigateToRootProject } from "store/navigation/navigation-action";
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { formatDate } from "common/formatters";
import { unionize, ofType, UnionOf } from "common/unionize";
import { VirtualMachineLogins } from 'models/virtual-machines';
import { FilterBuilder } from "services/api/filter-builder";
import { ListResults } from "services/common-service/common-service";
import { dialogActions } from 'store/dialog/dialog-actions';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { PermissionLevel } from "models/permission";
import { deleteResources, updateResources } from 'store/resources/resources-actions';
import { Participant } from "views-components/sharing-dialog/participant-select";
import { initialize, reset } from "redux-form";
import { getUserDisplayName, UserResource } from "models/user";

export const virtualMachinesActions = unionize({
    SET_REQUESTED_DATE: ofType<string>(),
    SET_VIRTUAL_MACHINES: ofType<ListResults<any>>(),
    SET_LOGINS: ofType<VirtualMachineLogins>(),
    SET_LINKS: ofType<ListResults<any>>()
});

export type VirtualMachineActions = UnionOf<typeof virtualMachinesActions>;

export const VIRTUAL_MACHINES_PANEL = 'virtualMachinesPanel';
export const VIRTUAL_MACHINE_ATTRIBUTES_DIALOG = 'virtualMachineAttributesDialog';
export const VIRTUAL_MACHINE_REMOVE_DIALOG = 'virtualMachineRemoveDialog';
export const VIRTUAL_MACHINE_ADD_LOGIN_DIALOG = 'virtualMachineAddLoginDialog';
export const VIRTUAL_MACHINE_ADD_LOGIN_FORM = 'virtualMachineAddLoginForm';
export const VIRTUAL_MACHINE_REMOVE_LOGIN_DIALOG = 'virtualMachineRemoveLoginDialog';

export const VIRTUAL_MACHINE_UPDATE_LOGIN_UUID_FIELD = 'uuid';
export const VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD = 'vmUuid';
export const VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD = 'user';
export const VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD = 'groups';
export const VIRTUAL_MACHINE_ADD_LOGIN_EXCLUDE = 'excludedPerticipants';

export const openUserVirtualMachines = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(navigateToUserVirtualMachines);
    };

export const openAdminVirtualMachines = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user;
        if (user && user.isAdmin) {
            dispatch<any>(navigateToAdminVirtualMachines);
        } else {
            dispatch<any>(navigateToRootProject);
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const openVirtualMachineAttributes = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const virtualMachineData = getState().virtualMachines.virtualMachines.items.find(it => it.uuid === uuid);
        dispatch(dialogActions.OPEN_DIALOG({ id: VIRTUAL_MACHINE_ATTRIBUTES_DIALOG, data: { virtualMachineData } }));
    };

const loadRequestedDate = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const date = services.virtualMachineService.getRequestedDate();
        dispatch(virtualMachinesActions.SET_REQUESTED_DATE(date));
    };

export const loadVirtualMachinesAdminData = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(loadRequestedDate());

        const virtualMachines = await services.virtualMachineService.list();
        dispatch(updateResources(virtualMachines.items));
        dispatch(virtualMachinesActions.SET_VIRTUAL_MACHINES(virtualMachines));


        const logins = await services.permissionService.list({
            filters: new FilterBuilder()
            .addIn('head_uuid', virtualMachines.items.map(item => item.uuid))
            .addEqual('name', PermissionLevel.CAN_LOGIN)
            .getFilters(),
            limit: 1000
        });
        dispatch(updateResources(logins.items));
        dispatch(virtualMachinesActions.SET_LINKS(logins));

        const users = await services.userService.list({
            filters: new FilterBuilder()
            .addIn('uuid', logins.items.map(item => item.tailUuid))
            .getFilters(),
            count: "none", // Necessary for federated queries
            limit: 1000
        });
        dispatch(updateResources(users.items));

        const getAllLogins = await services.virtualMachineService.getAllLogins();
        dispatch(virtualMachinesActions.SET_LOGINS(getAllLogins));
    };

export const loadVirtualMachinesUserData = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(loadRequestedDate());
        const user = getState().auth.user;
        const virtualMachines = await services.virtualMachineService.list();
        const virtualMachinesUuids = virtualMachines.items.map(it => it.uuid);
        const links = await services.linkService.list({
            filters: new FilterBuilder()
                .addIn("head_uuid", virtualMachinesUuids)
                .addEqual("tail_uuid", user?.uuid)
                .getFilters()
        });
        dispatch(virtualMachinesActions.SET_VIRTUAL_MACHINES(virtualMachines));
        dispatch(virtualMachinesActions.SET_LINKS(links));
    };

export const openAddVirtualMachineLoginDialog = (vmUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        // Get login permissions of vm
        const virtualMachines = await services.virtualMachineService.list();
        dispatch(updateResources(virtualMachines.items));
        const logins = await services.permissionService.list({
            filters: new FilterBuilder()
            .addIn('head_uuid', virtualMachines.items.map(item => item.uuid))
            .addEqual('name', PermissionLevel.CAN_LOGIN)
            .getFilters()
        });
        dispatch(updateResources(logins.items));

        dispatch(initialize(VIRTUAL_MACHINE_ADD_LOGIN_FORM, {
                [VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD]: vmUuid,
                [VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD]: [],
            }));
        dispatch(dialogActions.OPEN_DIALOG( {id: VIRTUAL_MACHINE_ADD_LOGIN_DIALOG, data: {excludedParticipants: logins.items.map(it => it.tailUuid)}} ));
    }

export const openEditVirtualMachineLoginDialog = (permissionUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const login = await services.permissionService.get(permissionUuid);
        const user = await services.userService.get(login.tailUuid);
        dispatch(initialize(VIRTUAL_MACHINE_ADD_LOGIN_FORM, {
                [VIRTUAL_MACHINE_UPDATE_LOGIN_UUID_FIELD]: permissionUuid,
                [VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD]: {name: getUserDisplayName(user, true, true), uuid: login.tailUuid},
                [VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD]: login.properties.groups,
            }));
        dispatch(dialogActions.OPEN_DIALOG( {id: VIRTUAL_MACHINE_ADD_LOGIN_DIALOG, data: {updating: true}} ));
    }

export interface AddLoginFormData {
    [VIRTUAL_MACHINE_UPDATE_LOGIN_UUID_FIELD]: string;
    [VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD]: string;
    [VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD]: Participant;
    [VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD]: string[];
}


export const addUpdateVirtualMachineLogin = ({uuid, vmUuid, user, groups}: AddLoginFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        let userResource: UserResource | undefined = undefined;
        try {
            // Get user
            userResource = await services.userService.get(user.uuid, false);
        } catch (e) {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Failed to get user details.", hideDuration: 2000, kind: SnackbarKind.ERROR }));
                return;
        }
        try {
            if (uuid) {
                const permission = await services.permissionService.update(uuid, {
                    tailUuid: userResource.uuid,
                    name: PermissionLevel.CAN_LOGIN,
                    properties: {
                        username: userResource.username,
                        groups,
                    }
                });
                dispatch(updateResources([permission]));
            } else {
                const permission = await services.permissionService.create({
                    headUuid: vmUuid,
                    tailUuid: userResource.uuid,
                    name: PermissionLevel.CAN_LOGIN,
                    properties: {
                        username: userResource.username,
                        groups,
                    }
                });
                dispatch(updateResources([permission]));
            }

            dispatch(reset(VIRTUAL_MACHINE_ADD_LOGIN_FORM));
            dispatch(dialogActions.CLOSE_DIALOG({ id: VIRTUAL_MACHINE_ADD_LOGIN_DIALOG }));
            dispatch<any>(loadVirtualMachinesAdminData());

            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `Permission updated`,
                kind: SnackbarKind.SUCCESS
            }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const openRemoveVirtualMachineLoginDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: VIRTUAL_MACHINE_REMOVE_LOGIN_DIALOG,
            data: {
                title: 'Remove login permission',
                text: 'Are you sure you want to remove this permission?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeVirtualMachineLogin = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            await services.permissionService.delete(uuid);
            dispatch<any>(deleteResources([uuid]));

            dispatch<any>(loadVirtualMachinesAdminData());

            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `Login permission removed`,
                kind: SnackbarKind.SUCCESS
            }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const saveRequestedDate = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const date = formatDate((new Date()).toISOString());
        services.virtualMachineService.saveRequestedDate(date);
        dispatch<any>(loadRequestedDate());
    };

export const openRemoveVirtualMachineDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: VIRTUAL_MACHINE_REMOVE_DIALOG,
            data: {
                title: 'Remove virtual machine',
                text: 'Are you sure you want to remove this virtual machine?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeVirtualMachine = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        await services.virtualMachineService.delete(uuid);
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        dispatch<any>(loadVirtualMachinesAdminData());
    };

const virtualMachinesBindedActions = bindDataExplorerActions(VIRTUAL_MACHINES_PANEL);

export const loadVirtualMachinesPanel = () =>
    (dispatch: Dispatch) => {
        dispatch(virtualMachinesBindedActions.REQUEST_ITEMS());
    };
