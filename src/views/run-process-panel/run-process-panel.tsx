// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { RunProcessPanelRootDataProps, RunProcessPanelRootActionProps, RunProcessPanelRoot } from 'views/run-process-panel/run-process-panel-root';
import { goToStep, runProcess, searchWorkflows, openSetWorkflowDialog } from 'store/run-process-panel/run-process-panel-actions';
import { WorkflowResource } from 'models/workflow';

const mapStateToProps = ({ runProcessPanel }: RootState): RunProcessPanelRootDataProps => {
    return {
        workflows: runProcessPanel.searchWorkflows,
        currentStep: runProcessPanel.currentStep,
        selectedWorkflow: runProcessPanel.selectedWorkflow
    };
};

const mapDispatchToProps = (dispatch: Dispatch): RunProcessPanelRootActionProps => ({
    onSetStep: (step: number) => {
        dispatch<any>(goToStep(step));
    },
    onSetWorkflow: (workflow: WorkflowResource) => {
        dispatch<any>(openSetWorkflowDialog(workflow));
    },
    runProcess: () => {
        dispatch<any>(runProcess);
    },
    onSearch: (term: string) => {
        dispatch<any>(searchWorkflows(term));
    }
});

export const RunProcessPanel = connect(mapStateToProps, mapDispatchToProps)(RunProcessPanelRoot);