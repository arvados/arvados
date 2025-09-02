// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";

export const RECENT_WF_RUNS_ID = "recentWorkflowRuns";
export const recentWorkflowRunsActions = bindDataExplorerActions(RECENT_WF_RUNS_ID);

export const loadRecentWorkflows = () => (dispatch: Dispatch) => {
    dispatch(recentWorkflowRunsActions.REQUEST_ITEMS());
}
