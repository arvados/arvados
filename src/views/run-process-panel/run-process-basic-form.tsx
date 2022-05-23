// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { reduxForm, Field } from 'redux-form';
import { Grid } from '@material-ui/core';
import { TextField } from 'components/text-field/text-field';
import { ProjectInput } from 'views/run-process-panel/inputs/project-input';
import { PROCESS_NAME_VALIDATION } from 'validators/validators';

export const RUN_PROCESS_BASIC_FORM = 'runProcessBasicForm';

export interface RunProcessBasicFormData {
    name: string;
    description: string;
    ownerUuid?: string;
}
export const RunProcessBasicForm =
    reduxForm<RunProcessBasicFormData>({
        form: RUN_PROCESS_BASIC_FORM
    })(() =>
        <form>
            <Grid container spacing={32}>
                <Grid item xs={12} md={6}>
                    <Field
                        name='name'
                        component={TextField as any}
                        label="Enter a new name for run process"
                        required
                        validate={PROCESS_NAME_VALIDATION} />
                </Grid>
                <Grid item xs={12} md={6}>
                    <Field
                        name='description'
                        component={TextField as any}
                        label="Enter a description for run process" />
                </Grid>
                <Grid item xs={12} md={6}>
                    <Field
                        name='ownerUuid'
                        component={ProjectInput as any}
                        label="Project to run the process in" />
                </Grid>
            </Grid>
        </form>);
