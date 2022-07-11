// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProcessLogsPanel } from './process-logs-panel';
import { ProcessLogsPanelAction, processLogsPanelActions } from './process-logs-panel-actions';

const initialState: ProcessLogsPanel = {
    filters: [],
    selectedFilter: '',
    logs: { '': [] },
};

export const processLogsPanelReducer = (state = initialState, action: ProcessLogsPanelAction): ProcessLogsPanel =>
    processLogsPanelActions.match(action, {
        RESET_PROCESS_LOGS_PANEL: () => initialState,
        INIT_PROCESS_LOGS_PANEL: ({ filters, logs }) => ({
            filters,
            logs,
            selectedFilter: filters[0] || '',
        }),
        SET_PROCESS_LOGS_PANEL_FILTER: selectedFilter => ({
            ...state,
            selectedFilter
        }),
        ADD_PROCESS_LOGS_PANEL_ITEM: ({ logType, log }) => {
            const filters = state.filters.indexOf(logType) > -1
                ? state.filters
                : [...state.filters, logType];
            const currentLogs = state.logs[logType] || [];
            const logsOfType = [...currentLogs, log];
            const logs = { ...state.logs, [logType]: logsOfType };
            return { ...state, logs, filters };
        },
        default: () => state,
    });
