// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { propertiesActions } from 'store/properties/properties-actions';
import { getProperty } from 'store/properties/properties';
import { navigateToRunProcess, navigateTo } from 'store/navigation/navigation-action';
import {
    goToStep,
    runProcessPanelActions,
    loadPresets,
    getWorkflowRunnerSettings
} from 'store/run-process-panel/run-process-panel-actions';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { initialize } from 'redux-form';
import { RUN_PROCESS_BASIC_FORM } from 'views/run-process-panel/run-process-basic-form';
import { RUN_PROCESS_INPUTS_FORM } from 'views/run-process-panel/run-process-inputs-form';
import { RUN_PROCESS_ADVANCED_FORM } from 'views/run-process-panel/run-process-advanced-form';
import { getResource } from 'store/resources/resources';
import { ProjectResource } from 'models/project';
import { UserResource } from 'models/user';
import { getWorkflowInputs, parseWorkflowDefinition } from 'models/workflow';

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

export const openRunProcess = (workflowUuid: string, ownerUuid?: string, name?: string, inputObj?: { [key: string]: any }) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const response = await services.workflowService.list();
        dispatch(runProcessPanelActions.SET_WORKFLOWS(response.items));

        const workflows = getState().runProcessPanel.searchWorkflows;
        const workflow = workflows.find(workflow => workflow.uuid === workflowUuid);
        if (workflow) {
            dispatch<any>(navigateToRunProcess);
            dispatch<any>(goToStep(1));
            dispatch(runProcessPanelActions.SET_STEP_CHANGED(true));
            dispatch(runProcessPanelActions.SET_SELECTED_WORKFLOW(workflow));
            dispatch<any>(loadPresets(workflow.uuid));

            dispatch(initialize(RUN_PROCESS_ADVANCED_FORM, getWorkflowRunnerSettings(workflow)));
            let owner;
            if (ownerUuid) {
                // Must be writable.
                owner = getResource<ProjectResource | UserResource>(ownerUuid)(getState().resources);
                if (!owner || !owner.canWrite) {
                    owner = undefined;
                }
            }
            if (owner) {
                dispatch(runProcessPanelActions.SET_PROCESS_OWNER_UUID(owner.uuid));
            }

            dispatch(initialize(RUN_PROCESS_BASIC_FORM, { name, owner }));

            const definition = parseWorkflowDefinition(workflow);
            if (definition) {
                const inputs = getWorkflowInputs(definition);
                if (inputs) {
                    const values = inputs.reduce((values, input) => ({
                        ...values,
                        [input.id]: input.default,
                    }), {});
                    dispatch(initialize(RUN_PROCESS_INPUTS_FORM, values));
                }
            }

            if (inputObj) {
                dispatch(initialize(RUN_PROCESS_INPUTS_FORM, inputObj));
            }
        } else {
            dispatch<any>(snackbarActions.OPEN_SNACKBAR({ message: `You can't run this process` }));
        }
    };

export const getPublicUserUuid = (state: RootState) => {
    const prefix = state.auth.localCluster;
    return `${prefix}-tpzed-anonymouspublic`;
};
export const getPublicGroupUuid = (state: RootState) => {
    const prefix = state.auth.localCluster;
    return `${prefix}-j7d0g-anonymouspublic`;
};
export const getAllUsersGroupUuid = (state: RootState) => {
    const prefix = state.auth.localCluster;
    return `${prefix}-j7d0g-fffffffffffffff`;
};

export const showWorkflowDetails = (uuid: string) =>
    propertiesActions.SET_PROPERTY({ key: WORKFLOW_PANEL_DETAILS_UUID, value: uuid });

export const getWorkflowDetails = (state: RootState) => {
    const uuid = getProperty<string>(WORKFLOW_PANEL_DETAILS_UUID)(state.properties);
    const workflows = state.runProcessPanel.workflows;
    const workflow = workflows.find(workflow => workflow.uuid === uuid);
    return workflow || undefined;
};

export const deleteWorkflow = (workflowUuid: string, ownerUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(navigateTo(ownerUuid));
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        await services.workflowService.delete(workflowUuid);
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
    };
