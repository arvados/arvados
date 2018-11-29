// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { propertiesActions } from '~/store/properties/properties-actions';
import { getResource } from '../resources/resources';
import { getProperty } from '~/store/properties/properties';
import { WorkflowResource } from '~/models/workflow';
import { navigateToRunProcess } from '~/store/navigation/navigation-action';
import { goToStep, runProcessPanelActions } from '~/store/run-process-panel/run-process-panel-actions';

export const WORKFLOW_PANEL_ID = "workflowPanel";
const UUID_PREFIX_PROPERTY_NAME = 'uuidPrefix';
const WORKFLOW_PANEL_DETAILS_UUID = 'workflowPanelDetailsUuid';
export const workflowPanelActions = bindDataExplorerActions(WORKFLOW_PANEL_ID);

export const loadWorkflowPanel = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(workflowPanelActions.REQUEST_ITEMS());
        const response = await services.workflowService.list();
        dispatch(runProcessPanelActions.SET_WORKFLOWS(response.items));
    };

export const setUuidPrefix = (uuidPrefix: string) =>
    propertiesActions.SET_PROPERTY({ key: UUID_PREFIX_PROPERTY_NAME, value: uuidPrefix });

export const getUuidPrefix = (state: RootState) => {
    return state.properties.uuidPrefix;
};

export const openRunProcess = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {  
        const workflows = getState().runProcessPanel.searchWorkflows;
        const workflow = workflows.find(workflow => workflow.uuid === uuid);
        dispatch<any>(navigateToRunProcess);
        dispatch(runProcessPanelActions.RESET_RUN_PROCESS_PANEL()); 
        dispatch<any>(goToStep(1));
        dispatch(runProcessPanelActions.SET_STEP_CHANGED(true));
        dispatch(runProcessPanelActions.SET_SELECTED_WORKFLOW(workflow!));       
    };

export const getPublicUserUuid = (state: RootState) => {
    const prefix = getProperty<string>(UUID_PREFIX_PROPERTY_NAME)(state.properties);
    return `${prefix}-tpzed-anonymouspublic`;
};
export const getPublicGroupUuid = (state: RootState) => {
    const prefix = getProperty<string>(UUID_PREFIX_PROPERTY_NAME)(state.properties);
    return `${prefix}-j7d0g-anonymouspublic`;
};

export const showWorkflowDetails = (uuid: string) =>
    propertiesActions.SET_PROPERTY({ key: WORKFLOW_PANEL_DETAILS_UUID, value: uuid });

export const getWorkflowDetails = (state: RootState) => {
    const uuid = getProperty<string>(WORKFLOW_PANEL_DETAILS_UUID)(state.properties);
    return uuid ? getResource<WorkflowResource>(uuid)(state.resources) : undefined;
};
