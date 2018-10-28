// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, StyleRulesCallback, Divider } from '@material-ui/core';
import { Field, WrappedFieldProps, WrappedFieldArrayProps, FieldArray, FieldsProps } from 'redux-form';
import { PermissionSelect } from './permission-select';
import { WithStyles } from '@material-ui/core/styles';
import withStyles from '@material-ui/core/styles/withStyles';


export default () =>
    <FieldArray name='permissions' component={SharingManagementFieldArray} />;

const SharingManagementFieldArray = ({ fields }: WrappedFieldArrayProps<{ email: string }>) =>
    <div>
        {
            fields.map((field, index, fields) =>
                <PermissionManagementRow key={field} {...{ field, index, fields }} />)
        }
        <Divider />
    </div>;

const permissionManagementRowStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        padding: `${theme.spacing.unit}px 0`,
    }
});
const PermissionManagementRow = withStyles(permissionManagementRowStyles)(
    ({ field, index, fields, classes }: { field: string, index: number, fields: FieldsProps<{ email: string }> } & WithStyles<'root'>) =>
        <>
            <Divider />
            <Grid container alignItems='center' spacing={8} className={classes.root}>
                <Grid item xs={8}>
                    {fields.get(index).email}
                </Grid>
                <Grid item xs={4}>
                    <Field name={`${field}.permissions`} component={PermissionSelectComponent} />
                </Grid>
            </Grid>
        </>
);

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <PermissionSelect fullWidth disableUnderline {...input} />;
