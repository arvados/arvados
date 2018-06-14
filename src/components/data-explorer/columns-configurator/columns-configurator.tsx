// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WithStyles, StyleRulesCallback, Theme, withStyles, IconButton, Popover, Paper, List, Checkbox, ListItemText, ListItem, Typography, ListSubheader } from '@material-ui/core';
import MenuIcon from "@material-ui/icons/Menu";
import { Column } from '../column';
import { PopoverOrigin } from '@material-ui/core/Popover';

export interface ColumnsConfiguratorProps {
    columns: Array<Column<any>>;
    onColumnToggle: (column: Column<any>) => void;
}


class ColumnsConfigurator extends React.Component<ColumnsConfiguratorProps & WithStyles<CssRules>> {

    state = {
        anchorEl: undefined
    };

    transformOrigin: PopoverOrigin = {
        vertical: "top",
        horizontal: "right",
    };

    render() {
        const { columns, onColumnToggle, classes } = this.props;
        return (
            <>
                <IconButton onClick={this.handleClick}><MenuIcon /></IconButton>
                <Popover
                    anchorEl={this.state.anchorEl}
                    open={Boolean(this.state.anchorEl)}
                    onClose={this.handleClose}
                    transformOrigin={this.transformOrigin}
                    anchorOrigin={this.transformOrigin}
                >
                    <Paper>
                        <List dense>
                            <ListSubheader>
                                Configure table columns
                            </ListSubheader>
                            {
                                columns.map((column, index) => (
                                    <ListItem key={index} button onClick={() => onColumnToggle(column)}>
                                        <Checkbox disableRipple color="primary" checked={column.selected} className={classes.checkbox} />
                                        <ListItemText>{column.header}</ListItemText>
                                    </ListItem>

                                ))
                            }
                        </List>
                    </Paper>
                </Popover>
            </>
        );
    }

    handleClose = () => {
        this.setState({ anchorEl: undefined });
    }

    handleClick = (event: React.MouseEvent<HTMLElement>) => {
        this.setState({ anchorEl: event.currentTarget });
    }

}

type CssRules = "root" | "checkbox";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {},
    checkbox: {
        width: 24,
        height: 24
    }
});

export default withStyles(styles)(ColumnsConfigurator);
