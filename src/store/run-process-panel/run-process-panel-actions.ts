// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { ServiceRepository } from "~/services/services";
import { RootState } from '~/store/store';
import { WorkflowResource } from '~/models/workflow';
import { getFormValues } from 'redux-form';
import { RUN_PROCESS_BASIC_FORM, RunProcessBasicFormData } from '~/views/run-process-panel/run-process-basic-form';
import { RUN_PROCESS_INPUTS_FORM } from '~/views/run-process-panel/run-process-inputs-form';
import { WorkflowInputsData } from '~/models/workflow';
import { createWorkflowMounts } from '~/models/process';
import { ContainerRequestState } from '~/models/container-request';
import { navigateToProcess } from '../navigation/navigation-action';
import { RunProcessAdvancedFormData, RUN_PROCESS_ADVANCED_FORM } from '~/views/run-process-panel/run-process-advanced-form';
import { isItemNotInProject, isProjectOrRunProcessRoute } from '~/store/projects/project-create-actions';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';

export const runProcessPanelActions = unionize({
    SET_PROCESS_OWNER_UUID: ofType<string>(),
    SET_CURRENT_STEP: ofType<number>(),
    SET_STEP_CHANGED: ofType<boolean>(),
    SET_WORKFLOWS: ofType<WorkflowResource[]>(),
    SET_SELECTED_WORKFLOW: ofType<WorkflowResource>(),
    SEARCH_WORKFLOWS: ofType<string>(),
    RESET_RUN_PROCESS_PANEL: ofType<{}>(),
});

export interface RunProcessSecondStepDataFormProps {
    name: string;
    description: string;
}

export const SET_WORKFLOW_DIALOG = 'setWorkflowDialog';
export const RUN_PROCESS_SECOND_STEP_FORM_NAME = 'runProcessSecondStepFormName';

export type RunProcessPanelAction = UnionOf<typeof runProcessPanelActions>;

export const loadRunProcessPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            dispatch(setBreadcrumbs([{ label: 'Run Process' }]));
            dispatch(runProcessPanelActions.RESET_RUN_PROCESS_PANEL());
            const response = await services.workflowService.list();
            dispatch(runProcessPanelActions.SET_WORKFLOWS(response.items));
        } catch (e) {
            return;
        }
    };

export const openSetWorkflowDialog = (workflow: WorkflowResource) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const selectedWorkflow = getState().runProcessPanel.selectedWorkflow;
        const isStepChanged = getState().runProcessPanel.isStepChanged;
        if (isStepChanged && selectedWorkflow && selectedWorkflow.uuid !== workflow.uuid) {
            dispatch(dialogActions.OPEN_DIALOG({
                id: SET_WORKFLOW_DIALOG,
                data: {
                    title: 'Form will be cleared',
                    text: 'Changing a workflow will clean all input fields in next step.',
                    confirmButtonLabel: 'Change Workflow',
                    workflow
                }
            }));
        } else {
            dispatch<any>(setWorkflow(workflow, false));
        }
    };

export const setWorkflow = (workflow: WorkflowResource, isWorkflowChanged = true) =>
    (dispatch: Dispatch<any>, getState: () => RootState) => {
        const isStepChanged = getState().runProcessPanel.isStepChanged;
        if (isStepChanged && isWorkflowChanged) {
            dispatch(runProcessPanelActions.SET_STEP_CHANGED(false));
            dispatch(runProcessPanelActions.SET_SELECTED_WORKFLOW(workflow));
        }
        if (!isWorkflowChanged) {
            dispatch(runProcessPanelActions.SET_SELECTED_WORKFLOW(workflow));
        }
    };

export const goToStep = (step: number) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        if (step === 1) {
            dispatch(runProcessPanelActions.SET_STEP_CHANGED(true));
        }
        dispatch(runProcessPanelActions.SET_CURRENT_STEP(step));
    };

export const runProcess = async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    const state = getState();
    const basicForm = getFormValues(RUN_PROCESS_BASIC_FORM)(state) as RunProcessBasicFormData;
    const inputsForm = getFormValues(RUN_PROCESS_INPUTS_FORM)(state) as WorkflowInputsData;
    const advancedForm = getFormValues(RUN_PROCESS_ADVANCED_FORM)(state) as RunProcessAdvancedFormData;
    const userUuid = getState().auth.user!.uuid;
    const router = getState();
    const properties = getState().properties;
    const { processOwnerUuid, selectedWorkflow } = state.runProcessPanel;
    if (selectedWorkflow) {
        const newProcessData = {
            ownerUuid: isItemNotInProject(properties) || !isProjectOrRunProcessRoute(router) ? userUuid : processOwnerUuid,
            name: basicForm.name,
            description: basicForm.description,
            state: ContainerRequestState.COMMITTED,
            mounts: createWorkflowMounts(selectedWorkflow, normalizeInputKeys(inputsForm)),
            runtimeConstraints: {
                API: true,
                vcpus: 1,
                ram: 1073741824,
            },
            containerImage: 'arvados/jobs',
            cwd: '/var/spool/cwl',
            command: [
                'arvados-cwl-runner',
                '--local',
                '--api=containers',
                `--project-uuid=${processOwnerUuid}`,
                '/var/lib/cwl/workflow.json#main',
                '/var/lib/cwl/cwl.input.json'
            ],
            outputPath: '/var/spool/cwl',
            priority: 1,
            outputName: advancedForm && advancedForm.output ? advancedForm.output : undefined,
        };
        const newProcess = await services.containerRequestService.create(newProcessData);
        dispatch(navigateToProcess(newProcess.uuid));
    }
};

const normalizeInputKeys = (inputs: WorkflowInputsData): WorkflowInputsData =>
    Object.keys(inputs).reduce((normalizedInputs, key) => ({
        ...normalizedInputs,
        [key.split('/').slice(1).join('/')]: inputs[key],
    }), {});
export const searchWorkflows = (term: string) => runProcessPanelActions.SEARCH_WORKFLOWS(term);
