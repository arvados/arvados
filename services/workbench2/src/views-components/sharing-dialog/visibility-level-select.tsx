// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { MenuItem, Select, withStyles, StyleRulesCallback } from '@material-ui/core';
import Lock from '@material-ui/icons/Lock';
import People from '@material-ui/icons/People';
import Public from '@material-ui/icons/Public';
import { WithStyles } from '@material-ui/core/styles';
import { SelectProps } from '@material-ui/core/Select';
import { SelectItem } from './select-item';
import { VisibilityLevel } from 'store/sharing-dialog/sharing-dialog-types';


type VisibilityLevelSelectClasses = 'root';

const VisibilityLevelSelectStyles: StyleRulesCallback<VisibilityLevelSelectClasses> = theme => ({
    root: {
        marginLeft: theme.spacing.unit,
    }
});
export const VisibilityLevelSelect = withStyles(VisibilityLevelSelectStyles)(
    ({ classes, includePublic, ...props }: { includePublic: boolean } & SelectProps & WithStyles<VisibilityLevelSelectClasses>) =>
        <Select
            {...props}
            renderValue={renderPermissionItem}
            inputProps={{ classes }}>
            {includePublic && <MenuItem value={VisibilityLevel.PUBLIC}>
                {renderPermissionItem(VisibilityLevel.PUBLIC)}
            </MenuItem>}
            <MenuItem value={VisibilityLevel.ALL_USERS}>
                {renderPermissionItem(VisibilityLevel.ALL_USERS)}
            </MenuItem>
            <MenuItem value={VisibilityLevel.SHARED}>
                {renderPermissionItem(VisibilityLevel.SHARED)}
            </MenuItem>
            <MenuItem value={VisibilityLevel.PRIVATE}>
                {renderPermissionItem(VisibilityLevel.PRIVATE)}
            </MenuItem>
        </Select>);

const renderPermissionItem = (value: string) =>
    <SelectItem {...{ value, icon: getIcon(value) }} />;

const getIcon = (value: string) => {
    switch (value) {
        case VisibilityLevel.PUBLIC:
            return Public;
        case VisibilityLevel.ALL_USERS:
            return Public;
        case VisibilityLevel.SHARED:
            return People;
        case VisibilityLevel.PRIVATE:
            return Lock;
        default:
            return Lock;
    }
};
