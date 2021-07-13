// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WithStyles, StyleRulesCallback, withStyles, IconButton, Paper, List, Checkbox, ListItemText, ListItem, Tooltip } from '@material-ui/core';
import MenuIcon from "@material-ui/icons/Menu";
import { DataColumn } from '../data-table/data-column';
import { Popover } from "../popover/popover";
import { IconButtonProps } from '@material-ui/core/IconButton';
import { DataColumns } from '../data-table/data-table';
import { ArvadosTheme } from "common/custom-theme";

interface ColumnSelectorDataProps {
    columns: DataColumns<any>;
    onColumnToggle: (column: DataColumn<any>) => void;
}

type CssRules = "checkbox";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    checkbox: {
        width: 24,
        height: 24
    }
});

export type ColumnSelectorProps = ColumnSelectorDataProps & WithStyles<CssRules>;

export const ColumnSelector = withStyles(styles)(
    ({ columns, onColumnToggle, classes }: ColumnSelectorProps) =>
        <Popover triggerComponent={ColumnSelectorTrigger}>
            <Paper>
                <List dense>
                    {columns
                        .filter(column => column.configurable)
                        .map((column, index) =>
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
                        )}
                </List>
            </Paper>
        </Popover>
);

export const ColumnSelectorTrigger = (props: IconButtonProps) =>
    <Tooltip disableFocusListener title="Select columns">
        <IconButton {...props}>
            <MenuIcon aria-label="Select columns" />
        </IconButton>
    </Tooltip>;
