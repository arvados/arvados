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
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { dialogActions } from 'store/dialog/dialog-actions';
import { ResourceKind, Resource } from 'models/resource';
import { selectedToArray } from "components/multiselect-toolbar/MultiselectToolbar";
import { CommonResourceServiceError, getCommonResourceServiceError } from "services/common-service/common-resource-service";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";

export const WORKFLOW_PANEL_ID = "workflowPanel";
const UUID_PREFIX_PROPERTY_NAME = 'uuidPrefix';
const WORKFLOW_PANEL_DETAILS_UUID = 'workflowPanelDetailsUuid';
export const workflowPanelActions = bindDataExplorerActions(WORKFLOW_PANEL_ID);

export const WORKFLOW_PROCESSES_PANEL_ID = "workflowProcessesPanel";
export const workflowProcessesPanelActions = bindDataExplorerActions(WORKFLOW_PROCESSES_PANEL_ID);

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
        const listedWorkflow = workflows.find(workflow => workflow.uuid === workflowUuid);
        const workflow = listedWorkflow || (await services.workflowService.get(workflowUuid));
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

export const openRemoveWorkflowDialog =
(resource: ContextMenuResource, numOfWorkflows: Number) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const confirmationText =
        numOfWorkflows === 1
            ? "Are you sure you want to remove this workflow?"
            : `Are you sure you want to remove these ${numOfWorkflows} workflows?`;
    const titleText = numOfWorkflows === 1 ? "Remove workflow permanently" : "Remove workflows permanently";

    dispatch(
        dialogActions.OPEN_DIALOG({
            id: REMOVE_WORKFLOW_DIALOG,
            data: {
                title: titleText,
                text: confirmationText,
                confirmButtonLabel: "Remove",
                uuid: resource.uuid,
                resource,
            },
        })
    );
};

export const REMOVE_WORKFLOW_DIALOG = "removeWorkflowDialog";

export const removeWorkflowPermanently = (uuid: string, ownerUuid?: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const resource = getState().dialog.removeWorkflowDialog.data.resource;
    const checkedList = getState().multiselect.checkedList;

    const uuidsToRemove: string[] = resource.fromContextMenu ? [resource.uuid] : selectedToArray(checkedList);

    //if no items in checkedlist, default to normal context menu behavior
    if (!uuidsToRemove.length) uuidsToRemove.push(uuid);
    if(ownerUuid) dispatch<any>(navigateTo(ownerUuid));

    const workflowsToRemove = uuidsToRemove
        .map(uuid => getResource(uuid)(getState().resources) as Resource)
        .filter(resource => resource.kind === ResourceKind.WORKFLOW);

    for (const workflow of workflowsToRemove) {
        try {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Removing ...", kind: SnackbarKind.INFO }));
            await services.workflowService.delete(workflow.uuid);
            dispatch(projectPanelDataActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Removed.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.PERMISSION_ERROR_FORBIDDEN) {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: `Access denied`, hideDuration: 2000, kind: SnackbarKind.ERROR }));
            } else {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: `Deletion failed`, hideDuration: 2000, kind: SnackbarKind.ERROR }));
            }
        }
    }
};
