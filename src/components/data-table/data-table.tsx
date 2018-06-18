// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Table, TableBody, TableRow, TableCell, TableHead, StyleRulesCallback, Theme, WithStyles, withStyles, Typography } from '@material-ui/core';
import { DataColumn } from './data-column';

export interface DataTableProps<T> {
    items: T[];
    columns: Array<DataColumn<T>>;
    onItemClick?: (item: T) => void;
}

class DataTable<T> extends React.Component<DataTableProps<T> & WithStyles<CssRules>> {
    render() {
        const { items, columns, classes, onItemClick } = this.props;
        return <div className={classes.tableContainer}>
            {items.length > 0 ?
                <Table>
                    <TableHead>
                        <TableRow>
                            {columns
                                .filter(column => column.selected)
                                .map(({ name, renderHeader, key }, index) =>
                                    <TableCell key={key || index}>
                                        {renderHeader ? renderHeader() : name}
                                    </TableCell>
                                )}
                        </TableRow>
                    </TableHead>
                    <TableBody className={classes.tableBody}>
                        {items
                            .map((item, index) =>
                                <TableRow
                                    hover
                                    key={index}
                                    onClick={() => onItemClick && onItemClick(item)}>
                                    {columns
                                        .filter(column => column.selected)
                                        .map((column, index) => (
                                            <TableCell key={column.key || index}>
                                                {column.render(item)}
                                            </TableCell>
                                        ))}
                                </TableRow>
                            )}
                    </TableBody>
                </Table> : <Typography>No items</Typography>}
        </div>;
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

export default withStyles(styles)(DataTable);
