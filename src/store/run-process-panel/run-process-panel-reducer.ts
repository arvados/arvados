// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RunProcessPanelAction, runProcessPanelActions } from '~/store/run-process-panel/run-process-panel-actions';
import { WorkflowResource, CommandInputParameter, getWorkflowInputs, parseWorkflowDefinition } from '~/models/workflow';

interface RunProcessPanel {
    processOwnerUuid: string;
    currentStep: number;
    isStepChanged: boolean;
    workflows: WorkflowResource[];
    searchWorkflows: WorkflowResource[];
    selectedWorkflow: WorkflowResource | undefined;
    presets?: WorkflowResource[];
    selectedPreset?: WorkflowResource;
    inputs: CommandInputParameter[];
}

const initialState: RunProcessPanel = {
    processOwnerUuid: '',
    currentStep: 0,
    isStepChanged: false,
    workflows: [],
    selectedWorkflow: undefined,
    inputs: [],
    searchWorkflows: [],
};

export const runProcessPanelReducer = (state = initialState, action: RunProcessPanelAction): RunProcessPanel =>
    runProcessPanelActions.match(action, {
        SET_PROCESS_OWNER_UUID: processOwnerUuid => ({ ...state, processOwnerUuid }),
        SET_CURRENT_STEP: currentStep => ({ ...state, currentStep }),
        SET_STEP_CHANGED: isStepChanged => ({ ...state, isStepChanged }),
        SET_SELECTED_WORKFLOW: selectedWorkflow => ({
            ...state,
            selectedWorkflow,
            presets: undefined,
            selectedPreset: selectedWorkflow,
            inputs: getWorkflowInputs(parseWorkflowDefinition(selectedWorkflow)) || [],
        }),
        SET_WORKFLOW_PRESETS: presets => ({
            ...state,
            presets,
        }),
        SELECT_WORKFLOW_PRESET: selectedPreset => ({
            ...state,
            selectedPreset,
        }),
        SET_WORKFLOWS: workflows => ({ ...state, workflows, searchWorkflows: workflows }),
        SEARCH_WORKFLOWS: term => {
            const termRegex = new RegExp(term, 'i');
            return {
                ...state,
                searchWorkflows: state.workflows.filter(workflow => termRegex.test(workflow.name)),
            };
        },
        RESET_RUN_PROCESS_PANEL: () => ({ ...initialState, processOwnerUuid: state.processOwnerUuid }),
        default: () => state
    });