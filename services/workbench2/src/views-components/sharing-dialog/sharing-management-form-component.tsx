// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, Divider, IconButton, Typography, Tooltip } from '@material-ui/core';
import {
    Field,
    WrappedFieldProps,
    WrappedFieldArrayProps,
    FieldArray,
    FieldArrayFieldsProps,
    InjectedFormProps
} from 'redux-form';
import { PermissionSelect, formatPermissionLevel, parsePermissionLevel } from './permission-select';
import { WithStyles } from '@material-ui/core/styles';
import withStyles from '@material-ui/core/styles/withStyles';
import { CloseIcon } from 'components/icon/icon';
import { ArvadosTheme } from 'common/custom-theme';

export interface SaveProps {
    onSave: () => void;
}

const headerStyles: CustomStyleRulesCallback<'heading'> = (theme: ArvadosTheme) => ({
    heading: {
        fontSize: '1.25rem',
    }
});

export const SharingManagementFormComponent = withStyles(headerStyles)(
    ({ classes, onSave }: WithStyles<'heading'> & SaveProps & InjectedFormProps<{}, SaveProps>) =>
        <>
            <Typography className={classes.heading}>People with access</Typography>
            <FieldArray<{ onSave: () => void }> name='permissions' component={SharingManagementFieldArray as any} props={{ onSave }} />
        </>);

export default SharingManagementFormComponent;

const SharingManagementFieldArray = ({ fields, onSave }: { onSave: () => void } & WrappedFieldArrayProps<{ email: string }>) =>
    <div>{fields.map((field, index, fields) =>
        <PermissionManagementRow key={field} {...{ field, index, fields }} onSave={onSave} />)}
    </div>;

const permissionManagementRowStyles: CustomStyleRulesCallback<'root'> = theme => ({
    root: {
        padding: `${theme.spacing.unit}px 0`,
    }
});

const PermissionManagementRow = withStyles(permissionManagementRowStyles)(
    ({ field, index, fields, classes, onSave }: { field: string, index: number, fields: FieldArrayFieldsProps<{ email: string }>, onSave: () => void; } & WithStyles<'root'>) =>
        <>
            <Grid container alignItems='center' spacing={8} wrap='nowrap' className={classes.root}>
                <Grid item xs={7}>
                    <Typography noWrap variant='subtitle1'>{fields.get(index).email}</Typography>
                </Grid>
                <Grid item xs={1} container wrap='nowrap'>
                    <Tooltip title='Remove access'>
                        <IconButton onClick={() => { fields.remove(index); onSave(); }}>
                            <CloseIcon />
                        </IconButton>
                    </Tooltip>
                </Grid>
                <Grid item xs={4} container wrap='nowrap'>
                    <Field
                        name={`${field}.permissions` as string}
                        component={PermissionSelectComponent}
                        format={formatPermissionLevel}
                        parse={parsePermissionLevel}
                        onChange={onSave}
                    />
                    
                </Grid>
            </Grid>
            <Divider />
        </>
);

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <PermissionSelect fullWidth disableUnderline {...input} />;
