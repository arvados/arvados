// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from '~/store/store';
import { ServiceRepository } from "~/services/services";
import { navigateToUserVirtualMachines, navigateToAdminVirtualMachines, navigateToRootProject } from "~/store/navigation/navigation-action";
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { formatDate } from "~/common/formatters";
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { VirtualMachineLogins } from '~/models/virtual-machines';
import { FilterBuilder } from "~/services/api/filter-builder";
import { ListResults } from "~/services/common-service/common-service";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';

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
        dispatch(virtualMachinesActions.SET_VIRTUAL_MACHINES(virtualMachines));
        const getAllLogins = await services.virtualMachineService.getAllLogins();
        dispatch(virtualMachinesActions.SET_LOGINS(getAllLogins));
    };

export const loadVirtualMachinesUserData = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(loadRequestedDate());
        const virtualMachines = await services.virtualMachineService.list();
        const virtualMachinesUuids = virtualMachines.items.map(it => it.uuid);
        const links = await services.linkService.list({
            filters: new FilterBuilder()
                .addIn("headUuid", virtualMachinesUuids)
                .getFilters()
        });
        dispatch(virtualMachinesActions.SET_VIRTUAL_MACHINES(virtualMachines));
        dispatch(virtualMachinesActions.SET_LINKS(links));
    };

export const saveRequestedDate = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const date = formatDate((new Date).toISOString());
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
