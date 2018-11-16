// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from '~/store/store';
import { ServiceRepository } from "~/services/services";
import { navigateToVirtualMachines } from "../navigation/navigation-action";
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { formatDate } from "~/common/formatters";
import { unionize, ofType, UnionOf } from "~/common/unionize";

export const virtualMachinesAction = unionize({
    SET_REQUESTED_DATE: ofType<string>(),
});

export type VirtualMachineActions = UnionOf<typeof virtualMachinesAction>;

export const VIRTUAL_MACHINES_PANEL = 'virtualMachinesPanel';

export const openVirtualMachines = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const virtualMachines = await services.virtualMachineService.list();
        // const logins = await services.virtualMachineService.logins(virtualMachines.items[0].uuid);
        // const getAllLogins = await services.virtualMachineService.getAllLogins();
        console.log(virtualMachines);
        // console.log(logins);
        // console.log(getAllLogins);      
        dispatch<any>(navigateToVirtualMachines);
    };

export const loadRequestedDate = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const date = services.virtualMachineService.getRequestedDate();
        dispatch(virtualMachinesAction.SET_REQUESTED_DATE(date));
    };

export const saveRequestedDate = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const date = formatDate((new Date).toISOString());
        services.virtualMachineService.saveRequestedDate(date);
        dispatch<any>(loadRequestedDate());
    };

const virtualMachinesActions = bindDataExplorerActions(VIRTUAL_MACHINES_PANEL);

export const loadVirtualMachinesPanel = () =>
    (dispatch: Dispatch) => {
        dispatch(virtualMachinesActions.REQUEST_ITEMS());
    };
