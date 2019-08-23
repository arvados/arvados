// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Button } from '@material-ui/core';
import { RunProcessBasicForm, RUN_PROCESS_BASIC_FORM } from './run-process-basic-form';
import { RunProcessInputsForm } from '~/views/run-process-panel/run-process-inputs-form';
import { CommandInputParameter, WorkflowResource } from '~/models/workflow';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { isValid } from 'redux-form';
import { RUN_PROCESS_INPUTS_FORM } from './run-process-inputs-form';
import { RunProcessAdvancedForm, RUN_PROCESS_ADVANCED_FORM } from './run-process-advanced-form';
import { createStructuredSelector } from 'reselect';
import { WorkflowPresetSelect } from '~/views/run-process-panel/workflow-preset-select';
import { selectPreset } from '~/store/run-process-panel/run-process-panel-actions';

export interface RunProcessSecondStepFormDataProps {
    inputs: CommandInputParameter[];
    workflow?: WorkflowResource;
    presets?: WorkflowResource[];
    selectedPreset?: WorkflowResource;
    valid: boolean;
}

export interface RunProcessSecondStepFormActionProps {
    goBack: () => void;
    runProcess: () => void;
    onPresetChange: (preset: WorkflowResource) => void;
}

const selectedWorkflowSelector = (state: RootState) =>
    state.runProcessPanel.selectedWorkflow;

const presetsSelector = (state: RootState) =>
    state.runProcessPanel.presets;

const selectedPresetSelector = (state: RootState) =>
    state.runProcessPanel.selectedPreset;

const inputsSelector = (state: RootState) =>
    state.runProcessPanel.inputs;

const validSelector = (state: RootState) =>
    isValid(RUN_PROCESS_BASIC_FORM)(state) && isValid(RUN_PROCESS_INPUTS_FORM)(state) && isValid(RUN_PROCESS_ADVANCED_FORM)(state);

const mapStateToProps = createStructuredSelector({
    inputs: inputsSelector,
    valid: validSelector,
    workflow: selectedWorkflowSelector,
    presets: presetsSelector,
    selectedPreset: selectedPresetSelector,
});

export type RunProcessSecondStepFormProps = RunProcessSecondStepFormDataProps & RunProcessSecondStepFormActionProps;
export const RunProcessSecondStepForm = connect(mapStateToProps, { onPresetChange: selectPreset })(
    ({ inputs, workflow, selectedPreset, presets, onPresetChange, valid, goBack, runProcess }: RunProcessSecondStepFormProps) =>
        <Grid container spacing={16}>
            <Grid item xs={12}>
                <Grid container spacing={32}>
                    <Grid item xs={12} md={6}>
                        {workflow && selectedPreset && presets &&
                            < WorkflowPresetSelect
                                {...{ workflow, selectedPreset, presets, onChange: onPresetChange }} />
                        }
                    </Grid>
                </Grid>
                <RunProcessBasicForm />
                <RunProcessInputsForm inputs={inputs} />
                <RunProcessAdvancedForm />
            </Grid>
            <Grid item xs={12}>
                <Button color="primary" onClick={goBack}>
                    Back
                </Button>
                <Button disabled={!valid} variant="contained" color="primary" onClick={runProcess}>
                    Run Process
                </Button>
            </Grid>
        </Grid>);
