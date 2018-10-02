// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Button } from '@material-ui/core';
import { RunProcessBasicForm, RUN_PROCESS_BASIC_FORM } from './run-process-basic-form';
import { RunProcessInputsForm } from '~/views/run-process-panel/run-process-inputs-form';
import { CommandInputParameter } from '~/models/workflow';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { isValid } from 'redux-form';
import { RUN_PROCESS_INPUTS_FORM } from './run-process-inputs-form';
import { RunProcessAdvancedForm } from './run-process-advanced-form';

export interface RunProcessSecondStepFormDataProps {
    inputs: CommandInputParameter[];
    valid: boolean;
}

export interface RunProcessSecondStepFormActionProps {
    goBack: () => void;
    runProcess: () => void;
}

const mapStateToProps = (state: RootState): RunProcessSecondStepFormDataProps => ({
    inputs: state.runProcessPanel.inputs,
    valid: isValid(RUN_PROCESS_BASIC_FORM)(state) &&
        isValid(RUN_PROCESS_INPUTS_FORM)(state),
});

export type RunProcessSecondStepFormProps = RunProcessSecondStepFormDataProps & RunProcessSecondStepFormActionProps;
export const RunProcessSecondStepForm = connect(mapStateToProps)(
    ({ inputs, valid, goBack, runProcess }: RunProcessSecondStepFormProps) =>
        <Grid container spacing={16}>
            <Grid item xs={12}>
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
