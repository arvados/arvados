// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { MenuItem, Select, withStyles, StyleRulesCallback } from '@material-ui/core';
import RemoveRedEye from '@material-ui/icons/RemoveRedEye';
import Edit from '@material-ui/icons/Edit';
import Computer from '@material-ui/icons/Computer';
import { WithStyles } from '@material-ui/core/styles';
import { SelectProps } from '@material-ui/core/Select';
import { SelectItem } from './select-item';

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
    <SelectItem {...{ value, icon: getIcon(value) }} />;

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
