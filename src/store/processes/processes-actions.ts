// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { updateResources } from '~/store/resources/resources-actions';
import { FilterBuilder } from '~/services/api/filter-builder';
import { ContainerRequestResource } from '~/models/container-request';
import { Process } from './process';
import { dialogActions } from '~/store/dialog/dialog-actions';
import {snackbarActions, SnackbarKind} from '~/store/snackbar/snackbar-actions';
import { projectPanelActions } from '~/store/project-panel/project-panel-action';

export const loadProcess = (containerRequestUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<Process> => {
        const containerRequest = await services.containerRequestService.get(containerRequestUuid);
        dispatch<any>(updateResources([containerRequest]));
        if (containerRequest.containerUuid) {
            const container = await services.containerService.get(containerRequest.containerUuid);
            dispatch<any>(updateResources([container]));
            await dispatch<any>(loadSubprocesses(containerRequest.containerUuid));
            return { containerRequest, container };
        }
        return { containerRequest };
    };

export const loadSubprocesses = (containerUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const containerRequests = await dispatch<any>(loadContainerRequests(
            new FilterBuilder().addEqual('requestingContainerUuid', containerUuid).getFilters()
        )) as ContainerRequestResource[];

        const containerUuids: string[] = containerRequests.reduce((uuids, { containerUuid }) =>
            containerUuid
                ? [...uuids, containerUuid]
                : uuids, []);

        if (containerUuids.length > 0) {
            await dispatch<any>(loadContainers(
                new FilterBuilder().addIn('uuid', containerUuids).getFilters()
            ));
        }
    };

export const loadContainerRequests = (filters: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { items } = await services.containerRequestService.list({ filters });
        dispatch<any>(updateResources(items));
        return items;
    };

export const loadContainers = (filters: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { items } = await services.containerService.list({ filters });
        dispatch<any>(updateResources(items));
        return items;
    };

export const openRemoveProcessDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: REMOVE_PROCESS_DIALOG,
            data: {
                title: 'Remove process permanently',
                text: 'Are you sure you want to remove this process?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const REMOVE_PROCESS_DIALOG = 'removeProcessDialog';

export const removeProcessPermanently = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) =>{
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        await services.containerRequestService.delete(uuid);
        dispatch(projectPanelActions.REQUEST_ITEMS());
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
    };
        

