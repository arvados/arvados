// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { FilterBuilder, joinFilters } from 'services/api/filter-builder';
import { ProgressBarStatus, ProgressBarCounts } from 'components/subprocess-progress-bar/subprocess-progress-bar';
import { ProcessStatusFilter, buildProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';
import { Process } from 'store/processes/process';
import { ProjectResource } from 'models/project';
import { getResource } from 'store/resources/resources';
import { ContainerRequestResource } from 'models/container-request';
import { Resource } from 'models/resource';

export const SUBPROCESS_PANEL_ID = "subprocessPanel";
export const SUBPROCESS_ATTRIBUTES_DIALOG = 'subprocessAttributesDialog';
export const subprocessPanelActions = bindDataExplorerActions(SUBPROCESS_PANEL_ID);

export const loadSubprocessPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(subprocessPanelActions.REQUEST_ITEMS());
    };

/**
 * Holds a ProgressBarData status type and process count result
 */
type ProcessStatusCount = {
    status: keyof ProgressBarCounts;
    count: number;
};

/**
 * Associates each of the limited progress bar segment types with an array of
 * ProcessStatusFilterTypes to be combined when displayed
 */
type ProcessStatusMap = Record<keyof ProgressBarCounts, ProcessStatusFilter[]>;

const statusMap: ProcessStatusMap = {
        [ProcessStatusFilter.COMPLETED]: [ProcessStatusFilter.COMPLETED],
        [ProcessStatusFilter.RUNNING]: [ProcessStatusFilter.RUNNING],
        [ProcessStatusFilter.FAILED]: [ProcessStatusFilter.FAILED, ProcessStatusFilter.CANCELLED],
        [ProcessStatusFilter.QUEUED]: [ProcessStatusFilter.QUEUED, ProcessStatusFilter.ONHOLD],
};

/**
 * Utility type to hold a pair of associated progress bar status and process status
 */
type ProgressBarStatusPair = {
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

export const fetchProcessProgressBarStatus = (parentResourceUuid: string, typeFilter?: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<ProgressBarStatus | undefined> => {
        const resources = getState().resources;
        const parentResource = getResource<ProjectResource | ContainerRequestResource>(parentResourceUuid)(resources);

        const requestContainerStatusCount = async (fb: FilterBuilder) => {
            return await services.containerRequestService.list({
                limit: 0,
                offset: 0,
                filters: fb.getFilters(),
            });
        }

        let baseFilter: string = "";
        if (isContainerRequest(parentResource) && parentResource.containerUuid) {
            // Prevent CR without containerUuid from generating baseFilter
            baseFilter = new FilterBuilder().addEqual('requesting_container_uuid', parentResource.containerUuid).getFilters();
        } else if (parentResource && !isContainerRequest(parentResource)) {
            // isCR type narrowing needed since CR without container may fall through
            baseFilter = new FilterBuilder().addEqual('owner_uuid', parentResource.uuid).getFilters();
        }

        if (parentResource && baseFilter) {
            // Add type filters from consumers that want to sync progress stats with filters
            if (typeFilter) {
                baseFilter = joinFilters(baseFilter, typeFilter);
            }

            try {
                // Create return object
                let result: ProgressBarCounts = {
                    [ProcessStatusFilter.COMPLETED]: 0,
                    [ProcessStatusFilter.RUNNING]: 0,
                    [ProcessStatusFilter.FAILED]: 0,
                    [ProcessStatusFilter.QUEUED]: 0,
                }

                // Create array of promises that returns the status associated with the item count
                // Helps to make the requests simultaneously while preserving the association with the status key as a typed key
                const promises = (Object.keys(statusMap) as Array<keyof ProcessStatusMap>)
                    // Split statusMap into pairs of progress bar status and process status
                    .reduce((acc, curr) => [...acc, ...statusMap[curr].map(processStatus => ({barStatus: curr, processStatus}))], [] as ProgressBarStatusPair[])
                    .map(async (statusPair: ProgressBarStatusPair): Promise<ProcessStatusCount> => {
                        // For each status pair, request count and return bar status and count
                        const { barStatus, processStatus } = statusPair;
                        const filter = buildProcessStatusFilters(new FilterBuilder(baseFilter), processStatus);
                        const count = (await requestContainerStatusCount(filter)).itemsAvailable;
                        if (count === undefined) return Promise.reject();
                        return {status: barStatus, count};
                    });

                // Simultaneously requests each status count and apply them to the return object
                const results = await resolvePromisesSequentially(promises);
                results.forEach((singleResult) => {
                    result[singleResult.status] += singleResult.count;
                });

                // CR polling is handled in progress bar based on store updates
                // This bool triggers polling without causing a final fetch when disabled
                // The shouldPoll logic here differs slightly from shouldPollProcess:
                //   * Process gets websocket updates through the store so using isProcessRunning
                //     ignores Queued
                //   * In projects, we get no websocket updates on CR state changes so we treat
                //     Queued processes as running in order to let polling keep us up to date
                //     when anything transitions to Running. This also means that a project with
                //     CRs in a stopped state won't start polling if CRs are started elsewhere
                const shouldPollProject = isContainerRequest(parentResource)
                    ? false
                    : (result[ProcessStatusFilter.RUNNING] + result[ProcessStatusFilter.QUEUED]) > 0;

                return {counts: result, shouldPollProject};
            } catch (e) {
                return undefined;
            }
        }
        return undefined;
    };

async function resolvePromisesSequentially<T>(promises: Promise<T>[]) {
    const results: T[] = [];

    for (const promise of promises) {
        // Yield control to the event loop before awaiting the promise
        await new Promise(resolve => setTimeout(resolve, 0));
        results.push(await promise);
    }

    return results;
}