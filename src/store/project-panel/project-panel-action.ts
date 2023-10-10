// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";
import { propertiesActions } from "store/properties/properties-actions";
import { RootState } from "store/store";
import { getProperty } from "store/properties/properties";
import { loadProject } from "store/workbench/workbench-actions";
import { projectPanelActions } from "store/data-explorer/data-explorer-action";

export const PROJECT_PANEL_ID = "projectPanel";
export const PROJECT_PANEL_CURRENT_UUID = "projectPanelCurrentUuid";
export const IS_PROJECT_PANEL_TRASHED = "isProjectPanelTrashed";

export const openProjectPanel = (projectUuid: string) => async (dispatch: Dispatch) => {
    await dispatch<any>(loadProject(projectUuid));
    dispatch(propertiesActions.SET_PROPERTY({ key: PROJECT_PANEL_CURRENT_UUID, value: projectUuid }));
    dispatch(projectPanelActions.RESET_EXPLORER_SEARCH_VALUE());
    dispatch(projectPanelActions.REQUEST_ITEMS());
};

export const getProjectPanelCurrentUuid = (state: RootState) => getProperty<string>(PROJECT_PANEL_CURRENT_UUID)(state.properties);

export const setIsProjectPanelTrashed = (isTrashed: boolean) => propertiesActions.SET_PROPERTY({ key: IS_PROJECT_PANEL_TRASHED, value: isTrashed });
