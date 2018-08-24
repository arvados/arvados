// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";
import { propertiesActions } from "~/store/properties/properties-actions";
import { Dispatch } from 'redux';
import { ServiceRepository } from "~/services/services";
import { RootState } from '~/store/store';
import { getProperty } from "~/store/properties/properties";
export const PROJECT_PANEL_ID = "projectPanel";
export const PROJECT_PANEL_CURRENT_UUID = "projectPanelCurrentUuid";
export const projectPanelActions = bindDataExplorerActions(PROJECT_PANEL_ID);

export const openProjectPanel = (projectUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(propertiesActions.SET_PROPERTY({ key: PROJECT_PANEL_CURRENT_UUID, value: projectUuid }));
        dispatch(projectPanelActions.REQUEST_ITEMS());
    };

export const getProjectPanelCurrentUuid = (state: RootState) => getProperty(PROJECT_PANEL_CURRENT_UUID)(state.properties);

