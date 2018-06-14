// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WithStyles, StyleRulesCallback, Theme, withStyles, IconButton, Paper, List, Checkbox, ListItemText, ListItem, Typography, ListSubheader } from '@material-ui/core';
import MenuIcon from "@material-ui/icons/Menu";
import { Column, isColumnConfigurable } from '../column';
import { PopoverOrigin } from '@material-ui/core/Popover';
import Popover from "../../popover/popover";
import { IconButtonProps } from '@material-ui/core/IconButton';

export interface ColumnsConfiguratorProps {
    columns: Array<Column<any>>;
    onColumnToggle: (column: Column<any>) => void;
}

const ColumnsConfigurator: React.SFC<ColumnsConfiguratorProps & WithStyles<CssRules>> = ({ columns, onColumnToggle, classes }) => {
    return (
        <Popover triggerComponent={ColumnsConfiguratorTrigger}>
            <Paper>
                <List dense>
                    {
                        columns
                            .filter(isColumnConfigurable)
                            .map((column, index) => (
                                <ListItem key={index} button onClick={() => onColumnToggle(column)}>
                                    <Checkbox disableRipple color="primary" checked={column.selected} className={classes.checkbox} />
                                    <ListItemText>{column.header}</ListItemText>
                                </ListItem>
                            ))
                    }
                </List>
            </Paper>
        </Popover>
    );
};

const ColumnsConfiguratorTrigger: React.SFC<IconButtonProps> = (props) => (
    <IconButton {...props}>
        <MenuIcon />
    </IconButton>
);

type CssRules = "root" | "checkbox";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {},
    checkbox: {
        width: 24,
        height: 24
    }
});

export default withStyles(styles)(ColumnsConfigurator);
