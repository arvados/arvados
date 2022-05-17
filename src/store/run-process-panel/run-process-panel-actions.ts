// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { unionize, ofType, UnionOf } from "common/unionize";
import { ServiceRepository } from "services/services";
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { WorkflowResource, WorkflowRunnerResources, getWorkflow, getWorkflowInputs, parseWorkflowDefinition } from 'models/workflow';
import { getFormValues, initialize } from 'redux-form';
import { RUN_PROCESS_BASIC_FORM, RunProcessBasicFormData } from 'views/run-process-panel/run-process-basic-form';
import { RUN_PROCESS_INPUTS_FORM } from 'views/run-process-panel/run-process-inputs-form';
import { WorkflowInputsData } from 'models/workflow';
import { createWorkflowMounts } from 'models/process';
import { ContainerRequestState } from 'models/container-request';
import { navigateTo } from '../navigation/navigation-action';
import {
    RunProcessAdvancedFormData, RUN_PROCESS_ADVANCED_FORM, VCPUS_FIELD,
    KEEP_CACHE_RAM_FIELD, RAM_FIELD, RUNTIME_FIELD, OUTPUT_FIELD, RUNNER_IMAGE_FIELD
} from 'views/run-process-panel/run-process-advanced-form';
import { dialogActions } from 'store/dialog/dialog-actions';
import { setBreadcrumbs } from 'store/breadcrumbs/breadcrumbs-actions';

export const runProcessPanelActions = unionize({
    SET_PROCESS_PATHNAME: ofType<string>(),
    SET_PROCESS_OWNER_UUID: ofType<string>(),
    SET_CURRENT_STEP: ofType<number>(),
    SET_STEP_CHANGED: ofType<boolean>(),
    SET_WORKFLOWS: ofType<WorkflowResource[]>(),
    SET_SELECTED_WORKFLOW: ofType<WorkflowResource>(),
    SET_WORKFLOW_PRESETS: ofType<WorkflowResource[]>(),
    SELECT_WORKFLOW_PRESET: ofType<WorkflowResource>(),
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

export const getWorkflowRunnerSettings = (workflow: WorkflowResource) => {
    const advancedFormValues = {};
    Object.assign(advancedFormValues, DEFAULT_ADVANCED_FORM_VALUES);

    const wf = getWorkflow(parseWorkflowDefinition(workflow));
    const hints = wf ? wf.hints : undefined;
    if (hints) {
        const resc = hints.find(item => item.class === 'http://arvados.org/cwl#WorkflowRunnerResources') as WorkflowRunnerResources | undefined;
        if (resc) {
            if (resc.ramMin) { advancedFormValues[RAM_FIELD] = resc.ramMin * (1024 * 1024); }
            if (resc.coresMin) { advancedFormValues[VCPUS_FIELD] = resc.coresMin; }
            if (resc.keep_cache) { advancedFormValues[KEEP_CACHE_RAM_FIELD] = resc.keep_cache * (1024 * 1024); }
            if (resc.acrContainerImage) { advancedFormValues[RUNNER_IMAGE_FIELD] = resc.acrContainerImage; }
        }
    }
    return advancedFormValues;
};

export const setWorkflow = (workflow: WorkflowResource, isWorkflowChanged = true) =>
    (dispatch: Dispatch<any>, getState: () => RootState) => {
        const isStepChanged = getState().runProcessPanel.isStepChanged;

        const advancedFormValues = getWorkflowRunnerSettings(workflow);

        if (isStepChanged && isWorkflowChanged) {
            dispatch(runProcessPanelActions.SET_STEP_CHANGED(false));
            dispatch(runProcessPanelActions.SET_SELECTED_WORKFLOW(workflow));
            dispatch<any>(loadPresets(workflow.uuid));
            dispatch(initialize(RUN_PROCESS_BASIC_FORM, { name: workflow.name }));
            dispatch(initialize(RUN_PROCESS_ADVANCED_FORM, advancedFormValues));
        }
        if (!isWorkflowChanged) {
            dispatch(runProcessPanelActions.SET_SELECTED_WORKFLOW(workflow));
            dispatch<any>(loadPresets(workflow.uuid));
            dispatch(initialize(RUN_PROCESS_BASIC_FORM, { name: workflow.name }));
            dispatch(initialize(RUN_PROCESS_ADVANCED_FORM, advancedFormValues));
        }
    };

export const loadPresets = (workflowUuid: string) =>
    async (dispatch: Dispatch<any>, _: () => RootState, { workflowService }: ServiceRepository) => {
        const { items } = await workflowService.presets(workflowUuid);
        dispatch(runProcessPanelActions.SET_WORKFLOW_PRESETS(items));
    };

export const selectPreset = (preset: WorkflowResource) =>
    (dispatch: Dispatch<any>) => {
        dispatch(runProcessPanelActions.SELECT_WORKFLOW_PRESET(preset));
        const inputs = getWorkflowInputs(parseWorkflowDefinition(preset)) || [];
        const values = inputs.reduce((values, input) => ({
            ...values,
            [input.id]: input.default,
        }), {});
        dispatch(initialize(RUN_PROCESS_INPUTS_FORM, values));
    };

export const goToStep = (step: number) =>
    (dispatch: Dispatch) => {
        if (step === 1) {
            dispatch(runProcessPanelActions.SET_STEP_CHANGED(true));
        }
        dispatch(runProcessPanelActions.SET_CURRENT_STEP(step));
    };

export const runProcess = async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    const state = getState();
    const basicForm = getFormValues(RUN_PROCESS_BASIC_FORM)(state) as RunProcessBasicFormData;
    const inputsForm = getFormValues(RUN_PROCESS_INPUTS_FORM)(state) as WorkflowInputsData;
    const userUuid = getUserUuid(getState());
    if (!userUuid) { return; }
    const { processOwnerUuid, selectedWorkflow } = state.runProcessPanel;
    const ownerUUid = processOwnerUuid ? processOwnerUuid : userUuid;
    if (selectedWorkflow) {
        const advancedForm = getFormValues(RUN_PROCESS_ADVANCED_FORM)(state) as RunProcessAdvancedFormData || getWorkflowRunnerSettings(selectedWorkflow);
        const newProcessData = {
            ownerUuid: ownerUUid,
            name: basicForm.name,
            description: basicForm.description,
            state: ContainerRequestState.COMMITTED,
            mounts: createWorkflowMounts(selectedWorkflow, normalizeInputKeys(inputsForm)),
            runtimeConstraints: {
                API: true,
                vcpus: advancedForm[VCPUS_FIELD],
                ram: (advancedForm[KEEP_CACHE_RAM_FIELD] + advancedForm[RAM_FIELD]),
            },
            schedulingParameters: {
                max_run_time: advancedForm[RUNTIME_FIELD]
            },
            containerImage: advancedForm[RUNNER_IMAGE_FIELD],
            cwd: '/var/spool/cwl',
            command: [
                'arvados-cwl-runner',
                '--api=containers',
                '--local',
                `--project-uuid=${ownerUUid}`,
                '/var/lib/cwl/workflow.json#main',
                '/var/lib/cwl/cwl.input.json'
            ],
            outputPath: '/var/spool/cwl',
            priority: 1,
            outputName: advancedForm[OUTPUT_FIELD] ? advancedForm[OUTPUT_FIELD] : `Output from ${basicForm.name}`,
            properties: {
                template_uuid: selectedWorkflow.uuid,
                workflowName: selectedWorkflow.name
            },
            useExisting: false
        };
        const newProcess = await services.containerRequestService.create(newProcessData);
        dispatch(navigateTo(newProcess.uuid));
    }
};

const DEFAULT_ADVANCED_FORM_VALUES: Partial<RunProcessAdvancedFormData> = {
    [VCPUS_FIELD]: 1,
    [RAM_FIELD]: 1073741824,
    [KEEP_CACHE_RAM_FIELD]: 268435456,
    [RUNNER_IMAGE_FIELD]: "arvados/jobs"
};

const normalizeInputKeys = (inputs: WorkflowInputsData): WorkflowInputsData =>
    Object.keys(inputs).reduce((normalizedInputs, key) => ({
        ...normalizedInputs,
        [key.split('/').slice(1).join('/')]: inputs[key],
    }), {});
export const searchWorkflows = (term: string) => runProcessPanelActions.SEARCH_WORKFLOWS(term);
