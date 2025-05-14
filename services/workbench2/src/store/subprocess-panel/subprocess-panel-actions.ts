// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { FilterBuilder, joinFilters } from 'services/api/filter-builder';
import { ProcessStatusFilter, buildProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';
import { Process } from 'store/processes/process';
import { ProjectResource } from 'models/project';
import { getResource } from 'store/resources/resources';
import { ContainerRequestResource } from 'models/container-request';
import { WorkflowResource } from 'models/workflow';
import { Resource, ResourceKind } from 'models/resource';
import { ALL_PROCESSES_PANEL_ID } from 'store/all-processes-panel/all-processes-panel-action';
import { SHARED_WITH_ME_PANEL_ID } from 'store/shared-with-me-panel/shared-with-me-panel-actions';

export const SUBPROCESS_PANEL_ID = "subprocessPanel";
export const SUBPROCESS_ATTRIBUTES_DIALOG = 'subprocessAttributesDialog';
export const subprocessPanelActions = bindDataExplorerActions(SUBPROCESS_PANEL_ID);

export const loadSubprocessPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(subprocessPanelActions.REQUEST_ITEMS());
    };

/**
 * Holds a Process status type and process count result
 */
type ProcessStatusCount = {
    status: keyof ProcessStatusCounts;
    count: string | null;
};

export type ProcessStatusCounts = {
    [ProcessStatusFilter.ALL]: string | null;
    [ProcessStatusFilter.COMPLETED]: string | null;
    [ProcessStatusFilter.RUNNING]: string | null;
    [ProcessStatusFilter.FAILED]: string | null;
    [ProcessStatusFilter.QUEUED]: string | null;
    [ProcessStatusFilter.ONHOLD]: string | null;
    [ProcessStatusFilter.CANCELLED]: string | null;
    [ProcessStatusFilter.DRAFT]: string | null;
};

/**
 * Associates each of the limited progress bar segment types with an array of
 * ProcessStatusFilterTypes to be combined when displayed
 */
type ProcessStatusMap = Record<keyof ProcessStatusCounts, ProcessStatusFilter[]>;

const statusMap: ProcessStatusMap = {
        [ProcessStatusFilter.ALL]: [ProcessStatusFilter.ALL],
        [ProcessStatusFilter.COMPLETED]: [ProcessStatusFilter.COMPLETED],
        [ProcessStatusFilter.RUNNING]: [ProcessStatusFilter.RUNNING],
        [ProcessStatusFilter.FAILED]: [ProcessStatusFilter.FAILED],
        [ProcessStatusFilter.CANCELLED]: [ProcessStatusFilter.CANCELLED],
        [ProcessStatusFilter.QUEUED]: [ProcessStatusFilter.QUEUED],
        [ProcessStatusFilter.ONHOLD]: [ProcessStatusFilter.ONHOLD],
        [ProcessStatusFilter.DRAFT]: [ProcessStatusFilter.DRAFT],
};

/**
 * Utility type to hold a pair of associated progress bar status and process status
 */
type ProcessStatusPair = {
    barStatus: keyof ProcessStatusMap;
    processStatus: ProcessStatusFilter;
};

/**
 * Type guard to distinguish Processes from other Resources
 * @param resource The item to check
 * @returns if the resource is a Process
 */
export const isProcess = <T extends Resource>(resource: T | Process | undefined): resource is Process => {
    return !!resource && 'containerRequest' in resource;
};

/**
 * Type guard to distinguish ContainerRequestResources from Resources
 * @param resource The item to check
 * @returns if the resource is a ContainerRequestResource
 */
const isContainerRequest = <T extends Resource>(resource: T | ContainerRequestResource | undefined): resource is ContainerRequestResource => {
    return !!resource && 'containerUuid' in resource;
};

export const fetchProcessStatusCounts = (parentResourceUuid: string, typeFilter?: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<ProcessStatusCounts | undefined> => {
        const resources = getState().resources;
        const parentResource = getResource<ProjectResource | ContainerRequestResource | WorkflowResource>(parentResourceUuid)(resources);

        const requestContainerStatusCount = async (fb: FilterBuilder) => {
            return await services.containerRequestService.list({
                limit: 0,
                offset: 0,
                filters: fb.getFilters(),
            });
        }

        const requestGroupsServiceCount = async (fb: FilterBuilder) => {
            return await services.groupsService.contents('', {
                limit: 0,
                count: 'exact',
                filters: fb.getFilters(),
                excludeHomeProject: true,
            });
        }

        let baseFilter: string = "";
        if (isContainerRequest(parentResource) && parentResource.containerUuid) {
            // Prevent CR without containerUuid from generating baseFilter
            baseFilter = new FilterBuilder().addEqual('requesting_container_uuid', parentResource.containerUuid).getFilters();
            // isCR type narrowing needed since CR without container may fall through
        } else if (parentResource?.kind === ResourceKind.WORKFLOW && !isContainerRequest(parentResource)) {
            baseFilter = new FilterBuilder().addEqual('properties.template_uuid', parentResource.uuid).getFilters();
        } else if (parentResource && !isContainerRequest(parentResource)) {
            baseFilter = new FilterBuilder().addEqual('owner_uuid', parentResource.uuid).getFilters();
        } else if (!isContainerRequest(parentResource) && isSharedWithMePanel(parentResourceUuid)) {
            const { auth } = getState();
            baseFilter = new FilterBuilder()
                .addIsA('uuid', 'arvados#containerRequest')
                .addEqual('requesting_container_uuid', null)
                .addDistinct('uuid', `${auth.config.uuidPrefix}-j7d0g-publicfavorites`)
                .addDistinct('owner_uuid', `${auth.user?.uuid}`)
                .getFilters();
        }

        if ((parentResource && baseFilter) || (isSharedWithMePanel(parentResourceUuid) && baseFilter) || isAllProcessesPanel(parentResourceUuid)) {
            // Add type filters from consumers that want to sync progress stats with filters
            if (typeFilter) {
                baseFilter = joinFilters(baseFilter, typeFilter);
            }

            try {
                // Create return object
                let result: ProcessStatusCounts = {
                    [ProcessStatusFilter.ALL]: null,
                    [ProcessStatusFilter.COMPLETED]: null,
                    [ProcessStatusFilter.RUNNING]: null,
                    [ProcessStatusFilter.FAILED]: null,
                    [ProcessStatusFilter.QUEUED]: null,
                    [ProcessStatusFilter.ONHOLD]: null,
                    [ProcessStatusFilter.CANCELLED]: null,
                    [ProcessStatusFilter.DRAFT]: null,
                }

                // Create array of promises that returns the status associated with the item count
                // Helps to make the requests simultaneously while preserving the association with the status key as a typed key
                const promises = (Object.keys(statusMap) as Array<keyof ProcessStatusMap>)
                    // Split statusMap into pairs of progress bar status and process status
                    .reduce((acc, curr) => [...acc, ...statusMap[curr].map(processStatus => ({barStatus: curr, processStatus}))], [] as ProcessStatusPair[])
                    .map(async (statusPair: ProcessStatusPair): Promise<ProcessStatusCount> => {
                        // For each status pair, request count and return bar status and count
                        const { barStatus, processStatus } = statusPair;
                        const filter = buildProcessStatusFilters(new FilterBuilder(baseFilter), processStatus);
                        const requestFunc = isSharedWithMePanel(parentResourceUuid) ? requestGroupsServiceCount : requestContainerStatusCount;
                        const count = (await requestFunc(filter))?.itemsAvailable?.toLocaleString();
                        if (count === undefined) return Promise.reject();
                        return {status: barStatus, count};
                    });

                // Simultaneously requests each status count and apply them to the return object
                const results = await resolvePromisesSequentially(promises);
                results.forEach((singleResult) => {
                    result[singleResult.status] = singleResult.count;
                });

                return result;
            } catch (e) {
                return undefined;
            }
        }
        return undefined;
    };

async function resolvePromisesSequentially<T>(promises: Promise<T>[]) {
    const results: T[] = [];

    for (const promise of promises) {
        try {
            // Yield control to the event loop before awaiting the promise
            await new Promise(resolve => setTimeout(resolve, 0));
            results.push(await promise);
        } catch (error) {
            console.error("Error while resolving promises sequentially", error);
        }
    }

    return results;
}

const isAllProcessesPanel = (parentResourceUuid: string) => parentResourceUuid === ALL_PROCESSES_PANEL_ID;
const isSharedWithMePanel = (parentResourceUuid: string) => parentResourceUuid === SHARED_WITH_ME_PANEL_ID;