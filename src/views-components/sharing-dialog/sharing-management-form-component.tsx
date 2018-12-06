// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, StyleRulesCallback, Divider, IconButton, Typography } from '@material-ui/core';
import {
    Field,
    WrappedFieldProps,
    WrappedFieldArrayProps,
    FieldArray,
    FieldArrayFieldsProps
} from 'redux-form';
import { PermissionSelect, formatPermissionLevel, parsePermissionLevel } from './permission-select';
import { WithStyles } from '@material-ui/core/styles';
import withStyles from '@material-ui/core/styles/withStyles';
import { CloseIcon } from '~/components/icon/icon';


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
    ({ field, index, fields, classes }: { field: string, index: number, fields: FieldArrayFieldsProps<{ email: string }> } & WithStyles<'root'>) =>
        <>
            <Divider />
            <Grid container alignItems='center' spacing={8} wrap='nowrap' className={classes.root}>
                <Grid item xs={8}>
                    <Typography noWrap variant='subheading'>{fields.get(index).email}</Typography>
                </Grid>
                <Grid item xs={4} container wrap='nowrap'>
                    <Field
                        name={`${field}.permissions`}
                        component={PermissionSelectComponent}
                        format={formatPermissionLevel}
                        parse={parsePermissionLevel} />
                    <IconButton onClick={() => fields.remove(index)}>
                        <CloseIcon />
                    </IconButton>
                </Grid>
            </Grid>
        </>
);

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <PermissionSelect fullWidth disableUnderline {...input} />;
