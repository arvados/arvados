// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, reset, InjectedFormProps } from 'redux-form';
import { PROJECT_CREATE_PROPERTIES_FORM_NAME, addPropertyToCreateProjectForm } from '~/store/projects/project-create-actions';
import { ResourcePropertiesFormData } from '~/views-components/resource-properties-form/resource-properties-form';
import { StyleRulesCallback, WithStyles, withStyles, Grid } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { PropertyKeyField } from '~/views-components/resource-properties-form/property-key-field';
import { PropertyValueField } from '~/views-components/resource-properties-form/property-value-field';
import { Button } from '~/views-components/resource-properties-form/resource-properties-form';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        paddingTop: theme.spacing.unit,
        margin: 0
    }
});

type CreateProjectPropertiesFormProps = InjectedFormProps<ResourcePropertiesFormData> & WithStyles<CssRules>;

const Form = withStyles(styles)(
    ({ handleSubmit, submitting, invalid, classes }: CreateProjectPropertiesFormProps) =>
        <Grid container spacing={16} className={classes.root}>
            <Grid item xs={5}>
                <PropertyKeyField />
            </Grid>
            <Grid item xs={5}>
                <PropertyValueField />
            </Grid>
            <Grid item xs={2}>
                <Button
                    disabled={invalid}
                    loading={submitting}
                    color='primary'
                    variant='contained'
                    onClick={handleSubmit}>
                    Add
                </Button>
            </Grid>
        </Grid>
);

export const CreateProjectPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: PROJECT_CREATE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch(addPropertyToCreateProjectForm(data));
        dispatch(reset(PROJECT_CREATE_PROPERTIES_FORM_NAME));
    }
})(Form);