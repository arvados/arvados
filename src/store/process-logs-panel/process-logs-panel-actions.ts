// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { ProcessLogs, getProcessLogsPanelCurrentUuid } from './process-logs-panel';
import { LogEventType } from 'models/log';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { Dispatch } from 'redux';
import { groupBy, min, reverse } from 'lodash';
import { LogResource } from 'models/log';
import { LogService } from 'services/log-service/log-service';
import { ResourceEventMessage } from 'websocket/resource-event-message';
import { getProcess } from 'store/processes/process';
import { FilterBuilder } from "services/api/filter-builder";
import { OrderBuilder } from "services/api/order-builder";
import { navigateTo } from 'store/navigation/navigation-action';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';

export const processLogsPanelActions = unionize({
    RESET_PROCESS_LOGS_PANEL: ofType<{}>(),
    INIT_PROCESS_LOGS_PANEL: ofType<{ filters: string[], logs: ProcessLogs }>(),
    SET_PROCESS_LOGS_PANEL_FILTER: ofType<string>(),
    ADD_PROCESS_LOGS_PANEL_ITEM: ofType<{ logType: string, log: string }>(),
});

export type ProcessLogsPanelAction = UnionOf<typeof processLogsPanelActions>;

export const setProcessLogsPanelFilter = (filter: string) =>
    processLogsPanelActions.SET_PROCESS_LOGS_PANEL_FILTER(filter);

export const initProcessLogsPanel = (processUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, { logService }: ServiceRepository) => {
        dispatch(processLogsPanelActions.RESET_PROCESS_LOGS_PANEL());
        const process = getProcess(processUuid)(getState().resources);
        const maxPageSize = getState().auth.config.clusterConfig.API.MaxItemsPerResponse;
        if (process && process.container) {
            const logResources = await loadContainerLogs(process.container.uuid, logService, maxPageSize);
            const initialState = createInitialLogPanelState(logResources);
            dispatch(processLogsPanelActions.INIT_PROCESS_LOGS_PANEL(initialState));
        }
    };

export const addProcessLogsPanelItem = (message: ResourceEventMessage<{ text: string }>) =>
    async (dispatch: Dispatch, getState: () => RootState, { logService }: ServiceRepository) => {
        if (PROCESS_PANEL_LOG_EVENT_TYPES.indexOf(message.eventType) > -1) {
            const uuid = getProcessLogsPanelCurrentUuid(getState().router);
            if (!uuid) { return }
            const process = getProcess(uuid)(getState().resources);
            if (!process) { return }
            const { containerRequest, container } = process;
            if (message.objectUuid === containerRequest.uuid
                || (container && message.objectUuid === container.uuid)) {
                dispatch(processLogsPanelActions.ADD_PROCESS_LOGS_PANEL_ITEM({
                    logType: ALL_FILTER_TYPE,
                    log: message.properties.text
                }));
                dispatch(processLogsPanelActions.ADD_PROCESS_LOGS_PANEL_ITEM({
                    logType: message.eventType,
                    log: message.properties.text
                }));
                if (MAIN_EVENT_TYPES.indexOf(message.eventType) > -1) {
                    dispatch(processLogsPanelActions.ADD_PROCESS_LOGS_PANEL_ITEM({
                        logType: MAIN_FILTER_TYPE,
                        log: message.properties.text
                    }));
                }
            }
        }
    };

const loadContainerLogs = async (containerUuid: string, logService: LogService, maxPageSize: number) => {
    const requestFilters = new FilterBuilder()
        .addEqual('object_uuid', containerUuid)
        .addIn('event_type', PROCESS_PANEL_LOG_EVENT_TYPES)
        .getFilters();
    const requestOrderAsc = new OrderBuilder<LogResource>()
        .addAsc('eventAt')
        .getOrder();
    const requestOrderDesc = new OrderBuilder<LogResource>()
        .addDesc('eventAt')
        .getOrder();
    const { items, itemsAvailable } = await logService.list({
        limit: maxPageSize,
        filters: requestFilters,
        order: requestOrderAsc,
    });

    // Request additional logs if necessary
    const remainingLogs = itemsAvailable - items.length;
    if (remainingLogs > 0) {
        const { items: itemsLast } = await logService.list({
            limit: min([maxPageSize, remainingLogs]),
            filters: requestFilters,
            order: requestOrderDesc,
            count: 'none',
        })
        if (remainingLogs - itemsLast.length > 0) {
            const snipLine = {
                ...items[items.length - 1],
                eventType: LogEventType.SNIP,
                properties: {
                    text: `================ 8< ================ 8< ========= Some log(s) were skipped ========= 8< ================ 8< ================`
                },
            }
            return [...items, snipLine, ...reverse(itemsLast)];
        }
        return [...items, ...reverse(itemsLast)];
    }
    return items;
};

const createInitialLogPanelState = (logResources: LogResource[]) => {
    const allLogs = logsToLines(logResources);
    const mainLogs = logsToLines(logResources.filter(
        e => MAIN_EVENT_TYPES.indexOf(e.eventType) > -1
    ));
    const groupedLogResources = groupBy(logResources, log => log.eventType);
    const groupedLogs = Object
        .keys(groupedLogResources)
        .reduce((grouped, key) => ({
            ...grouped,
            [key]: logsToLines(groupedLogResources[key])
        }), {});
    const filters = [
        MAIN_FILTER_TYPE,
        ALL_FILTER_TYPE,
        ...Object.keys(groupedLogs)
    ].filter(e => e !== LogEventType.SNIP);
    const logs = {
        [MAIN_FILTER_TYPE]: mainLogs,
        [ALL_FILTER_TYPE]: allLogs,
        ...groupedLogs
    };
    return { filters, logs };
};

const logsToLines = (logs: LogResource[]) =>
    logs.map(({ properties }) => properties.text);

export const navigateToLogCollection = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            await services.collectionService.get(uuid);
            dispatch<any>(navigateTo(uuid));
        } catch {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not request collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

const ALL_FILTER_TYPE = 'All logs';

const MAIN_FILTER_TYPE = 'Main logs';
const MAIN_EVENT_TYPES = [
    LogEventType.CRUNCH_RUN,
    LogEventType.STDERR,
    LogEventType.STDOUT,
    LogEventType.SNIP,
];

const PROCESS_PANEL_LOG_EVENT_TYPES = [
    LogEventType.ARV_MOUNT,
    LogEventType.CRUNCH_RUN,
    LogEventType.CRUNCHSTAT,
    LogEventType.DISPATCH,
    LogEventType.HOSTSTAT,
    LogEventType.NODE_INFO,
    LogEventType.STDERR,
    LogEventType.STDOUT,
    LogEventType.CONTAINER,
    LogEventType.KEEPSTORE,
    LogEventType.SNIP,
];
