// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Table, TableBody, TableRow, TableCell, TableHead, StyleRulesCallback, Theme, WithStyles, withStyles, Typography, Grid } from '@material-ui/core';
import { Column } from './column';
import ColumnsConfigurator from "./columns-configurator/columns-configurator";

export interface DataExplorerProps<T> {
    items: T[];
    columns: Array<Column<T>>;
    onColumnToggle: (column: Column<T>) => void;
    onItemClick?: (item: T) => void;
}

class DataExplorer<T> extends React.Component<DataExplorerProps<T> & WithStyles<CssRules>> {
    render() {
        const { items, columns, classes, onItemClick, onColumnToggle } = this.props;
        return (
            <div>
                <Grid container justify="flex-end">
                    <ColumnsConfigurator {...{ columns, onColumnToggle }} />
                </Grid>
                <div className={classes.tableContainer}>
                    {
                        items.length > 0 ? (
                            <Table>
                                <TableHead>
                                    <TableRow>
                                        {
                                            columns.filter(column => column.selected).map((column, index) => (
                                                <TableCell key={index}>{column.header}</TableCell>
                                            ))
                                        }
                                    </TableRow>
                                </TableHead>
                                <TableBody className={classes.tableBody}>
                                    {
                                        items.map((item, index) => (
                                            <TableRow key={index} hover onClick={() => onItemClick && onItemClick(item)}>
                                                {
                                                    columns.filter(column => column.selected).map((column, index) => (
                                                        <TableCell key={index}>
                                                            {column.render(item)}
                                                        </TableCell>
                                                    ))
                                                }
                                            </TableRow>
                                        ))
                                    }
                                </TableBody>
                            </Table>
                        ) : (
                                <Typography>No items</Typography>
                            )
                    }

                </div>
            </div>
        );
    }
}

type CssRules = "tableBody" | "tableContainer";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    tableContainer: {
        overflowX: 'auto'
    },
    tableBody: {
        background: theme.palette.background.paper
    }
});

export default withStyles(styles)(DataExplorer);
