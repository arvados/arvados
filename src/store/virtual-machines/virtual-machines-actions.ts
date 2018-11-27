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
import { VirtualMachinesLoginsResource } from '~/models/virtual-machines';
import { FilterBuilder } from "~/services/api/filter-builder";
import { ListResults } from "~/services/common-service/common-resource-service";

export const virtualMachinesActions = unionize({
    SET_REQUESTED_DATE: ofType<string>(),
    SET_VIRTUAL_MACHINES: ofType<ListResults<any>>(),
    SET_LOGINS: ofType<VirtualMachinesLoginsResource[]>(),
    SET_LINKS: ofType<ListResults<any>>()
});

export type VirtualMachineActions = UnionOf<typeof virtualMachinesActions>;

export const VIRTUAL_MACHINES_PANEL = 'virtualMachinesPanel';

export const openVirtualMachines = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(navigateToVirtualMachines);
    };

const loadRequestedDate = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const date = services.virtualMachineService.getRequestedDate();
        dispatch(virtualMachinesActions.SET_REQUESTED_DATE(date));
    };


export const loadVirtualMachinesData = () =>
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

const virtualMachinesBindedActions = bindDataExplorerActions(VIRTUAL_MACHINES_PANEL);

export const loadVirtualMachinesPanel = () =>
    (dispatch: Dispatch) => {
        dispatch(virtualMachinesBindedActions.REQUEST_ITEMS());
    };
