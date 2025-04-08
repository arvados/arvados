// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch } from 'redux';
import { Grid, Button } from '@mui/material';
import {
         RUN_PROCESS_BASIC_FORM,
         RUN_PROCESS_INPUTS_FORM,
    RUN_PROCESS_ADVANCED_FORM
} from 'store/run-process-panel/run-process-panel-actions';
import { RunProcessBasicForm } from './run-process-basic-form';
import { RunProcessAdvancedForm } from './run-process-advanced-form';
import { RunProcessInputsForm } from 'views/run-process-panel/run-process-inputs-form';
import { CommandInputParameter, WorkflowResource } from 'models/workflow';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { isValid, getFormSyncErrors } from 'redux-form';
import { createStructuredSelector } from 'reselect';
import { selectPreset } from 'store/run-process-panel/run-process-panel-actions';
import { getResource } from 'store/resources/resources';
import { ProjectResource } from 'models/project';
import { runProcessPanelActions } from 'store/run-process-panel/run-process-panel-actions';
import { getUserUuid } from 'common/getuser';

export interface RunProcessSecondStepFormDataProps {
    userUuid: string;
    inputs: CommandInputParameter[];
    workflow?: WorkflowResource;
    workflowOwner?: ProjectResource;
    defaultTargetProject?: ProjectResource;
    presets?: WorkflowResource[];
    selectedPreset?: WorkflowResource;
    valid: boolean;
}

export interface RunProcessSecondStepFormActionProps {
    goBack: () => void;
    runProcess: () => void;
    onPresetChange: (preset: WorkflowResource) => void;
    setProcessOwner: (ownerUuid: string) => void;
}

const selectedWorkflowSelector = (state: RootState) =>
    state.runProcessPanel.selectedWorkflow;

const presetsSelector = (state: RootState) =>
    state.runProcessPanel.presets;

const selectedPresetSelector = (state: RootState) =>
    state.runProcessPanel.selectedPreset;

const inputsSelector = (state: RootState) =>
    state.runProcessPanel.inputs;

const validSelector = (state: RootState) => {
    let isBasicFormValid = isValid(RUN_PROCESS_BASIC_FORM)(state);
    if (isBasicFormValid === false) {
        const syncErrors = getFormSyncErrors(RUN_PROCESS_BASIC_FORM)(state) as any;
        if (syncErrors && 'owner' in syncErrors && syncErrors.owner === true) {
            const defaultOwner = getResource<any>(state.runProcessPanel.processOwnerUuid)(state.resources);
            if (defaultOwner && defaultOwner.canWrite) {
                isBasicFormValid = true;
            }
        }
    }
    return isBasicFormValid && isValid(RUN_PROCESS_INPUTS_FORM)(state) && isValid(RUN_PROCESS_ADVANCED_FORM)(state);
}

const workflowOwnerSelector = (state: RootState) =>
    getResource<ProjectResource>(state.runProcessPanel.selectedWorkflow?.ownerUuid)(state.resources);

const defaultTargetProjectSelector = (state: RootState) =>
    getResource<ProjectResource>(state.runProcessPanel.processOwnerUuid)(state.resources);

const userUuidSelector = (state: RootState) =>
    getUserUuid(state);

const mapStateToProps = createStructuredSelector({
    userUuid: userUuidSelector,
    inputs: inputsSelector,
    valid: validSelector,
    workflow: selectedWorkflowSelector,
    workflowOwner: workflowOwnerSelector,
    defaultTargetProject: defaultTargetProjectSelector,
    presets: presetsSelector,
    selectedPreset: selectedPresetSelector,
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    setProcessOwner: (ownerUuid: string) => dispatch<any>(runProcessPanelActions.SET_PROCESS_OWNER_UUID(ownerUuid)),
    onPresetChange: selectPreset,
});

export type RunProcessSecondStepFormProps = RunProcessSecondStepFormDataProps & RunProcessSecondStepFormActionProps;
export const RunProcessSecondStepForm = connect(mapStateToProps, mapDispatchToProps)(
    ({ userUuid, inputs, workflow, workflowOwner, defaultTargetProject, selectedPreset, presets, onPresetChange, valid, goBack, runProcess, setProcessOwner }: RunProcessSecondStepFormProps) => {
        
        return <Grid container spacing={2} data-cy="new-process-panel">
                <Grid item xs={12}>
                    <RunProcessBasicForm workflow={workflow} />
                    <RunProcessInputsForm inputs={inputs} />
                    <RunProcessAdvancedForm />
                </Grid>
                <Grid item xs={12}>
                    <Button color="primary" onClick={goBack}>
                        Back
                    </Button>
                    <Button disabled={!valid} variant="contained" color="primary" onClick={runProcess}>
                        Run workflow
                    </Button>
                </Grid>
            </Grid>
        }
    );
