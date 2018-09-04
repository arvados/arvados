// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";
import { loadProcess } from '~/store/processes/processes-actions';
import { Dispatch } from 'redux';
import { ProcessStatus } from '~/store/processes/process';

export const procesPanelActions = unionize({
    INIT_PROCESS_PANEL_FILTERS: ofType<string[]>(),
    TOGGLE_PROCESS_PANEL_FILTER: ofType<string>(),
});

export type ProcessPanelAction = UnionOf<typeof procesPanelActions>;

export const toggleProcessPanelFilter = procesPanelActions.TOGGLE_PROCESS_PANEL_FILTER;

export const loadProcessPanel = (uuid: string) =>
    (dispatch: Dispatch) => {
        dispatch<any>(loadProcess(uuid));
        dispatch(initProcessPanelFilters);
    };

export const initProcessPanelFilters = procesPanelActions.INIT_PROCESS_PANEL_FILTERS([
    ProcessStatus.QUEUED,
    ProcessStatus.COMPLETED,
    ProcessStatus.FAILED,
    ProcessStatus.RUNNING,
    ProcessStatus.LOCKED,
    ProcessStatus.CANCELLED
]);
