// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProcessLogs, ProcessLogsPanel } from './process-logs-panel';
import { ProcessLogsPanelAction, processLogsPanelActions } from './process-logs-panel-actions';

const initialState: ProcessLogsPanel = {
    filters: [],
    selectedFilter: '',
    logs: {},
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
        ADD_PROCESS_LOGS_PANEL_ITEM: (groupedLogs: ProcessLogs) => {
            // Update filters
            const newFilters = Object.keys(groupedLogs).filter((logType) => (!state.filters.includes(logType)));
            const filters = [...state.filters, ...newFilters];

            // Append new log lines
            const logs = Object.keys(groupedLogs).reduce((acc, logType) => {
                if (Object.keys(acc).includes(logType)) {
                    // If log type exists, append lines and update lastByte
                    return {...acc, [logType]: {
                        lastByte: groupedLogs[logType].lastByte,
                        contents: [...acc[logType].contents, ...groupedLogs[logType].contents]
                    }};
                } else {
                    return {...acc, [logType]: groupedLogs[logType]};
                }
            }, state.logs);

            return { ...state, logs, filters };
        },
        default: () => state,
    });
