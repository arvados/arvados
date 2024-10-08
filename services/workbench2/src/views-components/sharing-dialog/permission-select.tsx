// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { MenuItem, Select } from '@mui/material';
import RemoveRedEye from '@mui/icons-material/RemoveRedEye';
import Edit from '@mui/icons-material/Edit';
import Computer from '@mui/icons-material/Computer';
import { SelectProps } from '@mui/material/Select';
import { SelectItem } from './select-item';
import { PermissionLevel } from '../../models/permission';

export enum PermissionSelectValue {
    READ = 'Read',
    WRITE = 'Write',
    MANAGE = 'Manage',
}

export const parsePermissionLevel = (value: PermissionSelectValue) => {
    switch (value) {
        case PermissionSelectValue.READ:
            return PermissionLevel.CAN_READ;
        case PermissionSelectValue.WRITE:
            return PermissionLevel.CAN_WRITE;
        case PermissionSelectValue.MANAGE:
            return PermissionLevel.CAN_MANAGE;
        default:
            return PermissionLevel.NONE;
    }
};

export const formatPermissionLevel = (value: PermissionLevel) => {
    switch (value) {
        case PermissionLevel.CAN_READ:
            return PermissionSelectValue.READ;
        case PermissionLevel.CAN_WRITE:
            return PermissionSelectValue.WRITE;
        case PermissionLevel.CAN_MANAGE:
            return PermissionSelectValue.MANAGE;
        default:
            return PermissionSelectValue.READ;
    }
};


export const PermissionSelect = (props: SelectProps) =>
    <Select
        variant="standard"
        {...props}
        disableUnderline
        data-cy="permission-select"
        renderValue={renderPermissionItem}>
        <MenuItem value={PermissionSelectValue.READ}>
            {renderPermissionItem(PermissionSelectValue.READ)}
        </MenuItem>
        <MenuItem value={PermissionSelectValue.WRITE}>
            {renderPermissionItem(PermissionSelectValue.WRITE)}
        </MenuItem>
        <MenuItem value={PermissionSelectValue.MANAGE}>
            {renderPermissionItem(PermissionSelectValue.MANAGE)}
        </MenuItem>
    </Select>;

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
