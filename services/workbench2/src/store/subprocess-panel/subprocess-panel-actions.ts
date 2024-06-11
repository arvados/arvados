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
import { Process, isProcessRunning } from 'store/processes/process';
import { ProjectResource } from 'models/project';

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

const isProcess = (resource: Process | ProjectResource | undefined): resource is Process => {
    return !!resource && 'containerRequest' in resource;
};

export const fetchProcessProgressBarStatus = (parentResource: Process | ProjectResource, typeFilter?: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<ProgressBarStatus | undefined> => {
        const requestContainerStatusCount = async (fb: FilterBuilder) => {
            return await services.containerRequestService.list({
                limit: 0,
                offset: 0,
                filters: fb.getFilters(),
            });
        }

        let baseFilter = "";
        if (isProcess(parentResource)) {
            baseFilter = new FilterBuilder().addEqual('requesting_container_uuid', parentResource.containerRequest.containerUuid).getFilters();
        } else {
            baseFilter = new FilterBuilder().addEqual('owner_uuid', parentResource.uuid).getFilters();
        }

        if (typeFilter) {
            baseFilter = joinFilters(baseFilter, typeFilter);
        }

        if (baseFilter) {
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
                        return {status: barStatus, count};
                    });

                // Simultaneously requests each status count and apply them to the return object
                (await Promise.all(promises)).forEach((singleResult) => {
                    result[singleResult.status] += singleResult.count;
                });

                let isRunning = result[ProcessStatusFilter.RUNNING] + result[ProcessStatusFilter.QUEUED] > 0;

                if (isProcess(parentResource)) {
                    isRunning = isProcessRunning(parentResource);
                }

                return {counts: result, isRunning};
            } catch (e) {
                return undefined;
            }
        } else {
            return undefined;
        }
    };
