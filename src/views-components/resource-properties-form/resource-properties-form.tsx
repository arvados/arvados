// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { Grid, withStyles, WithStyles } from '@material-ui/core';
import { PropertyKeyField, PROPERTY_KEY_FIELD_NAME } from './property-key-field';
import { PropertyValueField, PROPERTY_VALUE_FIELD_NAME } from './property-value-field';
import { ProgressButton } from '~/components/progress-button/progress-button';
import { GridClassKey } from '@material-ui/core/Grid';

export interface ResourcePropertiesFormData {
    [PROPERTY_KEY_FIELD_NAME]: string;
    [PROPERTY_VALUE_FIELD_NAME]: string;
}

export type ResourcePropertiesFormProps = InjectedFormProps<ResourcePropertiesFormData> & WithStyles<GridClassKey>;

export const ResourcePropertiesForm = ({ handleSubmit, submitting, invalid, classes }: ResourcePropertiesFormProps ) =>
    <form onSubmit={handleSubmit}>
        <Grid container spacing={16} classes={classes}>
            <Grid item xs>
                <PropertyKeyField />
            </Grid>
            <Grid item xs>
                <PropertyValueField />
            </Grid>
            <Grid item xs>
                <Button
                    disabled={invalid}
                    loading={submitting}
                    color='primary'
                    variant='contained'
                    type='submit'>
                    Add
                </Button>
            </Grid>
        </Grid>
    </form>;

const Button = withStyles(theme => ({
    root: { marginTop: theme.spacing.unit }
}))(ProgressButton);
