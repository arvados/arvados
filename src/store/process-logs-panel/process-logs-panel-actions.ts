// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";
import { ProcessLogs } from './process-logs-panel';
import { LogEventType } from '~/models/log';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { Dispatch } from 'redux';
import { FilterBuilder } from '~/common/api/filter-builder';
import { groupBy } from 'lodash';
import { loadProcess } from '~/store/processes/processes-actions';
import { OrderBuilder } from '~/common/api/order-builder';
import { LogResource } from '~/models/log';
import { LogService } from '~/services/log-service/log-service';

export const processLogsPanelActions = unionize({
    INIT_PROCESS_LOGS_PANEL: ofType<{ filters: string[], logs: ProcessLogs }>(),
    SET_PROCESS_LOGS_PANEL_FILTER: ofType<string>(),
    ADD_PROCESS_LOGS_PANEL_ITEM: ofType<{ logType: string, log: string }>(),
});

export type ProcessLogsPanelAction = UnionOf<typeof processLogsPanelActions>;

export const setProcessLogsPanelFilter = (filter: string) =>
     processLogsPanelActions.SET_PROCESS_LOGS_PANEL_FILTER(filter);

export const initProcessLogsPanel = (processUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, { logService }: ServiceRepository) => {
        const process = await dispatch<any>(loadProcess(processUuid));
        if (process.container) {
            const logResources = await loadContainerLogs(process.container.uuid, logService);
            const initialState = createInitialLogPanelState(logResources);
            dispatch(processLogsPanelActions.INIT_PROCESS_LOGS_PANEL(initialState));
        }
    };

const loadContainerLogs = async (containerUuid: string, logService: LogService) => {
    const requestFilters = new FilterBuilder()
        .addEqual('objectUuid', containerUuid)
        .addIn('eventType', PROCESS_PANEL_LOG_EVENT_TYPES)
        .getFilters();
    const requestOrder = new OrderBuilder<LogResource>()
        .addAsc('eventAt')
        .getOrder();
    const requestParams = {
        limit: MAX_AMOUNT_OF_LOGS,
        filters: requestFilters,
        order: requestOrder,
    };
    const { items } = await logService.list(requestParams);
    return items;
};

const createInitialLogPanelState = (logResources: LogResource[]) => {
    const allLogs = logResources.map(({ properties }) => properties.text);
    const groupedLogResources = groupBy(logResources, log => log.eventType);
    const groupedLogs = Object
        .keys(groupedLogResources)
        .reduce((grouped, key) => ({
            ...grouped,
            [key]: groupedLogResources[key].map(({ properties }) => properties.text)
        }), {});
    const filters = [SUMMARIZED_FILTER_TYPE, ...Object.keys(groupedLogs)];
    const logs = { [SUMMARIZED_FILTER_TYPE]: allLogs, ...groupedLogs };
    return { filters, logs };
};

const MAX_AMOUNT_OF_LOGS = 10000;

const SUMMARIZED_FILTER_TYPE = 'Summarized';

const PROCESS_PANEL_LOG_EVENT_TYPES = [
    LogEventType.ARV_MOUNT,
    LogEventType.CRUNCH_RUN,
    LogEventType.CRUNCHSTAT,
    LogEventType.DISPATCH,
    LogEventType.HOSTSTAT,
    LogEventType.NODE_INFO,
    LogEventType.STDERR,
    LogEventType.STDOUT,
];
