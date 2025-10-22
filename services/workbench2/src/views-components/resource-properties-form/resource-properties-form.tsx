// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { formValueSelector, InjectedFormProps } from 'redux-form';
import { Grid } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { PropertyKeyField, PROPERTY_KEY_FIELD_NAME, PROPERTY_KEY_FIELD_ID } from './property-key-field';
import { PropertyValueField, PROPERTY_VALUE_FIELD_NAME, PROPERTY_VALUE_FIELD_ID } from './property-value-field';
import { ProgressButton } from 'components/progress-button/progress-button';
import { GridClassKey } from '@mui/material/Grid';

const AddButton = withStyles(theme => ({
    root: { marginTop: theme.spacing(1) }
}))(ProgressButton);

const mapStateToProps = (state: RootState) => {
    return {
        applySelector: (selector) => selector(state, 'key', 'value', 'keyID', 'valueID')
    }
}

interface ApplySelector {
    applySelector: (selector) => any;
}

export interface ResourcePropertiesFormData {
    uuid: string;
    [PROPERTY_KEY_FIELD_NAME]: string;
    [PROPERTY_KEY_FIELD_ID]: string;
    [PROPERTY_VALUE_FIELD_NAME]: string;
    [PROPERTY_VALUE_FIELD_ID]: string;
    clearPropertyKeyOnSelect?: boolean;
}

type ResourcePropertiesFormProps = {uuid: string; clearPropertyKeyOnSelect?: boolean } & InjectedFormProps<ResourcePropertiesFormData, {uuid: string;}> & WithStyles<GridClassKey> & ApplySelector;

export const ResourcePropertiesForm = connect(mapStateToProps)(({ handleSubmit, change, submitting, invalid, classes, uuid, clearPropertyKeyOnSelect, applySelector,  ...props }: ResourcePropertiesFormProps ) => {
    change('uuid', uuid); // Sets the uuid field to the uuid of the resource.
    const propertyValue = applySelector(formValueSelector(props.form));
    return <form data-cy='resource-properties-form' onSubmit={handleSubmit}>
        <Grid container spacing={2} classes={classes}>
            <Grid item xs
            data-cy='key-input'>
                <PropertyKeyField clearPropertyKeyOnSelect />
            </Grid>
            <Grid item xs
            data-cy='value-input'>
                <PropertyValueField />
            </Grid>
            <Grid item>
                <AddButton
                    data-cy='property-add-btn'
                    disabled={invalid || !(propertyValue.key && propertyValue.value)}
                    loading={submitting}
                    color='primary'
                    variant='contained'
                    type='submit'>
                    Add
                </AddButton>
            </Grid>
        </Grid>
    </form>}
);
