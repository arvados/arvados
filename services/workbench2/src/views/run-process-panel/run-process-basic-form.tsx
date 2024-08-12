// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { reduxForm, Field, InjectedFormProps } from 'redux-form';
import { Grid, Typography, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core';
import { TextField } from 'components/text-field/text-field';
import { ProjectInput, ProjectCommandInputParameter } from 'views/run-process-panel/inputs/project-input';
import { PROCESS_NAME_VALIDATION } from 'validators/validators';
import { ProjectResource } from 'models/project';
import { UserResource } from 'models/user';
import { WorkflowResource } from 'models/workflow';
import { ArvadosTheme } from 'common/custom-theme';

export const RUN_PROCESS_BASIC_FORM = 'runProcessBasicForm';

export interface RunProcessBasicFormData {
    name: string;
    description: string;
    owner?: ProjectResource | UserResource;
}

interface RunProcessBasicFormProps {
    workflow?: WorkflowResource
}

type CssRules = 'name' | 'description';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    name: {
        overflow: "hidden",
        color: theme.customs.colors.greyD,
        fontSize: "1.875rem",
    },
    description: {
	marginBottom: "3em",
    }
});

export const RunProcessBasicForm =
    reduxForm<RunProcessBasicFormData, RunProcessBasicFormProps>({
        form: RUN_PROCESS_BASIC_FORM
    })(withStyles(styles)((props: InjectedFormProps<RunProcessBasicFormData, RunProcessBasicFormProps> & RunProcessBasicFormProps & WithStyles<CssRules>) =>
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
                    <ProjectInput required input={{
			id: "owner",
			label: "Project where the workflow will run"
                    } as ProjectCommandInputParameter}
				  options={{ showOnlyOwned: false, showOnlyWritable: true }} />
		</Grid>
		<Grid item xs={12} md={6}>
                    <Field
			name='description'
			component={TextField as any}
			label="Optional description" />
		</Grid>
		<Grid item xs={12}>
		    {props.workflow && <Typography className={props.classes.name}
						   data-cy='workflow-name'>{props.workflow.name}</Typography>}
		</Grid>
		<Grid item xs={12}>
		    {props.workflow && <Typography className={props.classes.description}
						   data-cy='workflow-description'
						   //dangerouslySetInnerHTML is ok here only if description is sanitized,
						   //which it is before it is loaded into the redux store
						   dangerouslySetInnerHTML={{ __html: props.workflow.description }}
				       />}
		</Grid>
	    </Grid>
	</form>));
