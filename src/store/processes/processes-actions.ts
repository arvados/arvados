// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { updateResources } from '~/store/resources/resources-actions';
import { FilterBuilder } from '~/services/api/filter-builder';
import { ContainerRequestResource } from '../../models/container-request';
import { Process } from './process';

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
