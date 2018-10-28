// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Field, WrappedFieldProps, FieldArray, WrappedFieldArrayProps } from 'redux-form';
import { Grid, FormControl, InputLabel } from '@material-ui/core';
import { PermissionSelect } from './permission-select';
import { PeopleSelect, Person } from './people-select';

export default () =>
    <Grid container spacing={8}>
        <Grid item xs={8}>
            <InvitedPeopleField />
        </Grid>
        <Grid item xs={4}>
            <PermissionSelectField />
        </Grid>
    </Grid>;

const InvitedPeopleField = () =>
    <FieldArray
        name='invitedPeople'
        component={InvitedPeopleFieldComponent} />;


const InvitedPeopleFieldComponent = ({ fields }: WrappedFieldArrayProps<Person>) =>
    <PeopleSelect
        items={fields.getAll() || []}
        onCreate={fields.push}
        onSelect={fields.push}
        onDelete={fields.remove} />;

const PermissionSelectField = () =>
    <Field
        name='permission'
        component={PermissionSelectComponent} />;

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <FormControl fullWidth>
        <InputLabel>Authorization</InputLabel>
        <PermissionSelect {...input} />
    </FormControl>;
