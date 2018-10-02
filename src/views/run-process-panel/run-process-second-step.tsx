// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { withStyles, WithStyles, StyleRulesCallback, Grid, Button } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { Field, reduxForm, InjectedFormProps } from 'redux-form';
import { TextField } from '~/components/text-field/text-field';
import { RunProcessSecondStepDataFormProps, RUN_PROCESS_SECOND_STEP_FORM_NAME } from '~/store/run-process-panel/run-process-panel-actions';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {

    }
});

export interface RunProcessSecondStepDataProps {

}

export interface RunProcessSecondStepActionProps {
    onSetStep: (step: number) => void;
    onRunProcess: (data: RunProcessSecondStepDataFormProps) => void;
}

type RunProcessSecondStepProps = RunProcessSecondStepDataProps 
    & RunProcessSecondStepActionProps 
    & WithStyles<CssRules> 
    & InjectedFormProps<RunProcessSecondStepDataFormProps>;

const RunProcessSecondStep = withStyles(styles)(
    ({ onSetStep, classes }: RunProcessSecondStepProps) =>
        <Grid container spacing={16}>
            <Grid item xs={12}>
                <form>
                    <Field
                        name='name'
                        component={TextField}
                        label="Enter a new name for run process" />
                    <Field
                        name='description'
                        component={TextField}
                        label="Enter a description for run process" />
                </form>
            </Grid>
            <Grid item xs={12}>
                <Button color="primary" onClick={() => onSetStep(0)}>
                    Back
                </Button>
                <Button variant="contained" color="primary">
                    Run Process
                </Button>
            </Grid>
        </Grid>
);

export const RunProcessSecondStepForm = reduxForm<RunProcessSecondStepDataFormProps>({
    form: RUN_PROCESS_SECOND_STEP_FORM_NAME
})(RunProcessSecondStep);