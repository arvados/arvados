// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WithStyles, StyleRulesCallback, Theme, withStyles, IconButton, Paper, List, Checkbox, ListItemText, ListItem } from '@material-ui/core';
import MenuIcon from "@material-ui/icons/Menu";
import { DataColumn, isColumnConfigurable } from '../data-table/data-column';
import Popover from "../popover/popover";
import { IconButtonProps } from '@material-ui/core/IconButton';

export interface ColumnSelectorProps {
    columns: Array<DataColumn<any>>;
    onColumnToggle: (column: DataColumn<any>) => void;
}

const ColumnSelector: React.SFC<ColumnSelectorProps & WithStyles<CssRules>> = ({ columns, onColumnToggle, classes }) =>
    <Popover triggerComponent={ColumnSelectorTrigger}>
        <Paper>
            <List dense>
                {columns
                    .filter(isColumnConfigurable)
                    .map((column, index) => (
                        <ListItem
                            button
                            key={index}
                            onClick={() => onColumnToggle(column)}>
                            <Checkbox
                                disableRipple
                                color="primary"
                                checked={column.selected}
                                className={classes.checkbox} />
                            <ListItemText>
                                {column.name}
                            </ListItemText>
                        </ListItem>
                    ))}
            </List>
        </Paper>
    </Popover>;

export const ColumnSelectorTrigger: React.SFC<IconButtonProps> = (props) =>
    <IconButton {...props}>
        <MenuIcon />
    </IconButton>;

type CssRules = "checkbox";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    checkbox: {
        width: 24,
        height: 24
    }
});

export default withStyles(styles)(ColumnSelector);
