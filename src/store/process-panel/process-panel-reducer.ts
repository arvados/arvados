// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProcessPanel } from '~/store/process-panel/process-panel';
import { ProcessPanelAction, processPanelActions } from '~/store/process-panel/process-panel-actions';

const initialState: ProcessPanel = {
    containerRequestUuid: "",
    filters: {}
};

export const processPanelReducer = (state = initialState, action: ProcessPanelAction): ProcessPanel =>
    processPanelActions.match(action, {
        SET_PROCESS_PANEL_CONTAINER_REQUEST_UUID: containerRequestUuid => ({
            ...state, containerRequestUuid
        }),
        SET_PROCESS_PANEL_FILTERS: statuses => {
            const filters = statuses.reduce((filters, status) => ({ ...filters, [status]: true }), {});
            return { ...state, filters };
        },
        TOGGLE_PROCESS_PANEL_FILTER: status => {
            const filters = { ...state.filters, [status]: !state.filters[status] };
            return { ...state, filters };
        },
        default: () => state,
    });
