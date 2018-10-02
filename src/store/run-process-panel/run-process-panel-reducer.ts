// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RunProcessPanelAction, runProcessPanelActions } from '~/store/run-process-panel/run-process-panel-actions';
import { WorkflowResource } from '~/models/workflow';

interface RunProcessPanel {
    currentStep: number;
    workflows: WorkflowResource[];
    searchWorkflows: WorkflowResource[];
    selectedWorkflow: WorkflowResource | undefined;
}

const initialState: RunProcessPanel = {
    currentStep: 0,
    workflows: [],
    searchWorkflows: [],
    selectedWorkflow: undefined
};

export const runProcessPanelReducer = (state = initialState, action: RunProcessPanelAction): RunProcessPanel =>
    runProcessPanelActions.match(action, {
        SET_CURRENT_STEP: currentStep => ({ ...state, currentStep }),
        SET_WORKFLOWS: workflows => ({ ...state, workflows, searchWorkflows: workflows }), 
        SET_SELECTED_WORKFLOW: selectedWorkflow => ({ ...state, selectedWorkflow }),
        SEARCH_WORKFLOWS: term => ({ ...state, searchWorkflows: state.workflows.filter(workflow => workflow.name.includes(term)) }),
        default: () => state
    });