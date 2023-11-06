// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { reduxForm, Field } from 'redux-form';
import { Grid } from '@material-ui/core';
import { TextField } from 'components/text-field/text-field';
import { ProjectInput, ProjectCommandInputParameter } from 'views/run-process-panel/inputs/project-input';
import { PROCESS_NAME_VALIDATION } from 'validators/validators';
import { ProjectResource } from 'models/project';
import { UserResource } from 'models/user';

export const RUN_PROCESS_BASIC_FORM = 'runProcessBasicForm';

export interface RunProcessBasicFormData {
    name: string;
    description: string;
    owner?: ProjectResource | UserResource;
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
                        label="Name for this workflow run"
                        required
                        validate={PROCESS_NAME_VALIDATION} />
                </Grid>
                <Grid item xs={12} md={6}>
                    <Field
                        name='description'
                        component={TextField as any}
                        label="Optional description of this workflow run" />
                </Grid>
                <Grid item xs={12} md={6}>
                    <ProjectInput required input={{
                        id: "owner",
                        label: "Project where the workflow will run"
                    } as ProjectCommandInputParameter}
                        options={{ showOnlyOwned: false, showOnlyWritable: true }} />
                </Grid>
            </Grid>
        </form>);
