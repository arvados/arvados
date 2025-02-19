// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { IconButton, Paper, List, Checkbox, ListItemText, ListItem, Tooltip } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import MenuIcon from "@mui/icons-material/Menu";
import { DataColumn, DataColumns } from '../data-table/data-column';
import { Popover } from "../popover/popover";
import { IconButtonProps } from '@mui/material/IconButton';
import { ArvadosTheme } from "common/custom-theme";

interface ColumnSelectorDataProps {
    columns: DataColumns<any, any>;
    onColumnToggle: (column: DataColumn<any, any>) => void;
    className?: string;
}

type CssRules = "checkbox" | "listItem" | "listItemText";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    checkbox: {
        width: 24,
        height: 24
    },
    listItem: {
        padding: 0,
    },
    listItemText: {
        paddingLeft: '4px',
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
                                className={classes.listItem}
                                data-cy={'column-selector-li'}
                                onClick={() => onColumnToggle(column)}>
                                <Checkbox
                                    disableRipple
                                    color="primary"
                                    checked={column.selected}
                                    className={classes.checkbox} />
                                <ListItemText
                                    className={classes.listItemText}>
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
        <IconButton {...props} size="large">
            <MenuIcon aria-label="Select columns" data-cy="column-selector-button" />
        </IconButton>
    </Tooltip>;
