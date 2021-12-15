// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { Grid, withStyles, WithStyles } from '@material-ui/core';
import { PropertyKeyField, PROPERTY_KEY_FIELD_NAME, PROPERTY_KEY_FIELD_ID } from './property-key-field';
import { PropertyValueField, PROPERTY_VALUE_FIELD_NAME, PROPERTY_VALUE_FIELD_ID } from './property-value-field';
import { ProgressButton } from 'components/progress-button/progress-button';
import { GridClassKey } from '@material-ui/core/Grid';

export interface ResourcePropertiesFormData {
    uuid: string;
    [PROPERTY_KEY_FIELD_NAME]: string;
    [PROPERTY_KEY_FIELD_ID]: string;
    [PROPERTY_VALUE_FIELD_NAME]: string;
    [PROPERTY_VALUE_FIELD_ID]: string;
}

export type ResourcePropertiesFormProps = {uuid: string; } & InjectedFormProps<ResourcePropertiesFormData, {uuid: string; }> & WithStyles<GridClassKey>;

export const ResourcePropertiesForm = ({ handleSubmit, change, submitting, invalid, classes, uuid }: ResourcePropertiesFormProps ) => {
    change('uuid', uuid); // Sets the uuid field to the uuid of the resource.
    return <form data-cy='resource-properties-form' onSubmit={handleSubmit}>
        <Grid container spacing={16} classes={classes}>
            <Grid item xs>
                <PropertyKeyField />
            </Grid>
            <Grid item xs>
                <PropertyValueField />
            </Grid>
            <Grid item xs>
                <Button
                    data-cy='property-add-btn'
                    disabled={invalid}
                    loading={submitting}
                    color='primary'
                    variant='contained'
                    type='submit'>
                    Add
                </Button>
            </Grid>
        </Grid>
    </form>};

export const Button = withStyles(theme => ({
    root: { marginTop: theme.spacing.unit }
}))(ProgressButton);
