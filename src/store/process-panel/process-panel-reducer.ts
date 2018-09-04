// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProcessPanel } from '~/store/process-panel/process-panel';
import { ProcessPanelAction, procesPanelActions } from '~/store/process-panel/process-panel-actions';

const initialState: ProcessPanel = {
    filters: {}
};

export const processPanelReducer = (state = initialState, action: ProcessPanelAction): ProcessPanel =>
    procesPanelActions.match(action, {
        INIT_PROCESS_PANEL_FILTERS: statuses => {
            const filters = statuses.reduce((filters, status) => ({ ...filters, [status]: true }), {});
            return { filters };
        },
        TOGGLE_PROCESS_PANEL_FILTER: status => {
            const filters = { ...state.filters, [status]: !state.filters[status] };
            return { filters };
        },
        default: () => state,
    });
    