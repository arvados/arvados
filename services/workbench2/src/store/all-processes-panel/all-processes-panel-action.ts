// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";

export const ALL_PROCESSES_PANEL_ID = "allProcessesPanel";
export const allProcessesPanelActions = bindDataExplorerActions(ALL_PROCESSES_PANEL_ID);

export const loadAllProcessesPanel = () => (dispatch: Dispatch) => {
    dispatch(allProcessesPanelActions.RESET_EXPLORER_SEARCH_VALUE());
    dispatch(allProcessesPanelActions.REQUEST_ITEMS());
}
