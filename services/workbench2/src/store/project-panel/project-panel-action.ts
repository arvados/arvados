// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { propertiesActions } from "store/properties/properties-actions";
import { loadProject } from "store/workbench/workbench-actions";
import { projectPanelRunActions, projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { PROJECT_PANEL_CURRENT_UUID, IS_PROJECT_PANEL_TRASHED } from "./project-panel";

/**
 * Project panel tab labels
 * This is used to associate the labels used to display tabs / determine default
 * project tab with the values stored in user preferences which also referece these values
 */
export const ProjectPanelTabLabels = {
    Overview: "Overview",
    DATA: "Data",
    RUNS: "Workflow Runs",
};

export const RootProjectPanelTabLabels = {
    DATA: "Data",
    RUNS: "Workflow Runs",
};

export const openProjectPanel = (projectUuid: string) => async (dispatch: Dispatch) => {
    // Pre-emptively set working as early as possible to avoid delay from loadProject codepath
    dispatch(projectPanelDataActions.SET_WORKING(true));
    dispatch(projectPanelRunActions.SET_WORKING(true));

    await dispatch<any>(loadProject(projectUuid));
    dispatch(propertiesActions.SET_PROPERTY({ key: PROJECT_PANEL_CURRENT_UUID, value: projectUuid }));

    dispatch(projectPanelDataActions.RESET_EXPLORER_SEARCH_VALUE());
    dispatch(projectPanelDataActions.REQUEST_ITEMS());

    dispatch(projectPanelRunActions.RESET_EXPLORER_SEARCH_VALUE());
    dispatch(projectPanelRunActions.REQUEST_ITEMS());
};

export const setIsProjectPanelTrashed = (isTrashed: boolean) => propertiesActions.SET_PROPERTY({ key: IS_PROJECT_PANEL_TRASHED, value: isTrashed });
