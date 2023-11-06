// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { FilterBuilder } from 'services/api/filter-builder';
import { ProgressBarData } from 'components/subprocess-progress-bar/subprocess-progress-bar';
import { ProcessStatusFilter, buildProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';
export const SUBPROCESS_PANEL_ID = "subprocessPanel";
export const SUBPROCESS_ATTRIBUTES_DIALOG = 'subprocessAttributesDialog';
export const subprocessPanelActions = bindDataExplorerActions(SUBPROCESS_PANEL_ID);

export const loadSubprocessPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(subprocessPanelActions.REQUEST_ITEMS());
    };

type ProcessStatusCount = {
    status: keyof ProgressBarData;
    count: number;
};

export const fetchSubprocessProgress = (requestingContainerUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<ProgressBarData | undefined> => {

        const requestContainerStatusCount = async (fb: FilterBuilder) => {
            return await services.containerRequestService.list({
                limit: 0,
                offset: 0,
                filters: fb.getFilters(),
            });
        }

        if (requestingContainerUuid) {
            try {
                const baseFilter = new FilterBuilder().addEqual('requesting_container_uuid', requestingContainerUuid).getFilters();

                // Create return object
                let result: ProgressBarData = {
                    [ProcessStatusFilter.COMPLETED]: 0,
                    [ProcessStatusFilter.RUNNING]: 0,
                    [ProcessStatusFilter.FAILED]: 0,
                    [ProcessStatusFilter.QUEUED]: 0,
                }

                // Create array of promises that returns the status associated with the item count
                // Helps to make the requests simultaneously while preserving the association with the status key as a typed key
                const promises = Object.keys(result).map(async (status: keyof ProgressBarData): Promise<ProcessStatusCount> => {
                    const filter = buildProcessStatusFilters(new FilterBuilder(baseFilter), status);
                    const count = (await requestContainerStatusCount(filter)).itemsAvailable;
                    return {status, count};
                });

                // Simultaneously requests each status count and apply them to the return object
                (await Promise.all(promises)).forEach((singleResult) => {
                    result[singleResult.status] = singleResult.count;
                });
                return result;
            } catch (e) {
                return undefined;
            }
        } else {
            return undefined;
        }
    };
