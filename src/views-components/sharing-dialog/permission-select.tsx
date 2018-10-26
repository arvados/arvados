// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { MenuItem, Grid, Select, withStyles, StyleRulesCallback } from '@material-ui/core';
import RemoveRedEye from '@material-ui/icons/RemoveRedEye';
import Edit from '@material-ui/icons/Edit';
import Computer from '@material-ui/icons/Computer';
import { WithStyles } from '@material-ui/core/styles';
import { SelectProps } from '@material-ui/core/Select';

export enum PermissionSelectValue {
    READ = 'Read',
    WRITE = 'Write',
    MANAGE = 'Manage',
}

type PermissionSelectClasses = 'value';

const PermissionSelectStyles: StyleRulesCallback<PermissionSelectClasses> = theme => ({
    value: {
        marginLeft: theme.spacing.unit,
    }
});

export const PermissionSelect = withStyles(PermissionSelectStyles)(
    ({ classes, ...props }: SelectProps & WithStyles<PermissionSelectClasses>) =>
        <Select
            {...props}
            renderValue={renderPermissionItem}
            inputProps={{ classes }}>
            <MenuItem value={PermissionSelectValue.READ}>
                {renderPermissionItem(PermissionSelectValue.READ)}
            </MenuItem>
            <MenuItem value={PermissionSelectValue.WRITE}>
                {renderPermissionItem(PermissionSelectValue.WRITE)}
            </MenuItem>
            <MenuItem value={PermissionSelectValue.MANAGE}>
                {renderPermissionItem(PermissionSelectValue.MANAGE)}
            </MenuItem>
        </Select>);

const renderPermissionItem = (value: string) =>
    <PermissionItem {...{ value }} />;

type PermissionItemClasses = 'value' | 'icon';

const permissionItemStyles: StyleRulesCallback<PermissionItemClasses> = theme => ({
    value: {
        marginLeft: theme.spacing.unit,
    },
    icon: {
       margin: `-${theme.spacing.unit / 2}px 0`,
    }
});

const PermissionItem = withStyles(permissionItemStyles)(
    ({ value, classes }: { value: string } & WithStyles<PermissionItemClasses>) => {
        const Icon = getIcon(value);
        return (
            <Grid container alignItems='center'>
                <Icon className={classes.icon} />
                <span className={classes.value}>
                    {value}
                </span>
            </Grid>);
    });

const getIcon = (value: string) => {
    switch (value) {
        case PermissionSelectValue.READ:
            return RemoveRedEye;
        case PermissionSelectValue.WRITE:
            return Edit;
        case PermissionSelectValue.MANAGE:
            return Computer;
        default:
            return Computer;
    }
};
