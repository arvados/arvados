// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Field, WrappedFieldProps } from 'redux-form';
import { Grid, Input, FormControl, FormHelperText, FormLabel, InputLabel, Chip } from '@material-ui/core';
import { ChipsInput } from '~/components/chips-input/chips-input';
import { identity } from 'lodash';
import { PermissionSelect } from './permission-select';
import { PeopleSelect } from './people-select';
import ChipInput from 'material-ui-chip-input';

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
    <Field
        name='invitedPeople'
        component={InvitedPeopleFieldComponent} />;


const InvitedPeopleFieldComponent = (props: WrappedFieldProps) =>
    <PeopleSelect suggestedPeople={[]} />;

const PermissionSelectField = () =>
    <Field
        name='permission'
        component={PermissionSelectComponent} />;

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <FormControl fullWidth>
        <InputLabel>Authorization</InputLabel>
        <PermissionSelect {...input} />
    </FormControl>;
