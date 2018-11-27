// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps, reduxForm } from 'redux-form';
import { Grid, Button } from '@material-ui/core';
import { PropertyKeyField, PROPERTY_KEY_FIELD_NAME } from './property-key-field';
import { PropertyValueField, PROPERTY_VALUE_FIELD_NAME } from './property-value-field';

export interface ResourcePropertiesFormData {
    [PROPERTY_KEY_FIELD_NAME]: string;
    [PROPERTY_VALUE_FIELD_NAME]: string;
}

export const ResourcePropertiesForm = reduxForm({ form: 'rpform' })(
    ({ handleSubmit }: InjectedFormProps) =>
        <form onSubmit={handleSubmit}>
            <Grid container spacing={16}>
                <Grid item xs>
                    <PropertyKeyField />
                </Grid>
                <Grid item xs>
                    <PropertyValueField />
                </Grid>
                <Grid item xs>
                    <Button variant='contained'>Add</Button>
                </Grid>
            </Grid>
        </form>);
