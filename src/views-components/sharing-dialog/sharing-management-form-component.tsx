// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
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
import { CloseIcon } from 'components/icon/icon';

const SharingManagementFormComponent = (props: { onSave: () => void; }) =>
    <FieldArray<{ onSave: () => void }> name='permissions' component={SharingManagementFieldArray as any} props={props} />;

export default SharingManagementFormComponent;

const SharingManagementFieldArray = ({ fields, onSave }: { onSave: () => void } & WrappedFieldArrayProps<{ email: string }>) =>
    <div>{fields.map((field, index, fields) =>
        <PermissionManagementRow key={field} {...{ field, index, fields }} onSave={onSave} />)}
        <Divider />
    </div>;

const permissionManagementRowStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        padding: `${theme.spacing.unit}px 0`,
    }
});

const PermissionManagementRow = withStyles(permissionManagementRowStyles)(
    ({ field, index, fields, classes, onSave }: { field: string, index: number, fields: FieldArrayFieldsProps<{ email: string }>, onSave: () => void; } & WithStyles<'root'>) =>
        <>
            <Divider />
            <Grid container alignItems='center' spacing={8} wrap='nowrap' className={classes.root}>
                <Grid item xs={8}>
                    <Typography noWrap variant='subtitle1'>{fields.get(index).email}</Typography>
                </Grid>
                <Grid item xs={4} container wrap='nowrap'>
                    <Field
                        name={`${field}.permissions` as string}
                        component={PermissionSelectComponent}
                        format={formatPermissionLevel}
                        parse={parsePermissionLevel}
                        onChange={onSave}
                    />
                    <IconButton onClick={() => { fields.remove(index); onSave(); }}>
                        <CloseIcon />
                    </IconButton>
                </Grid>
            </Grid>
        </>
);

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <PermissionSelect fullWidth disableUnderline {...input} />;
