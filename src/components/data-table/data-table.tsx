// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Table, TableBody, TableRow, TableCell, TableHead, StyleRulesCallback, Theme, WithStyles, withStyles, Typography } from '@material-ui/core';
import { DataColumn } from './data-column';

export type DataColumns<T> = Array<DataColumn<T>>;

export interface DataTableProps<T> {
    items: T[];
    columns: DataColumns<T>;
    onRowClick?: (event: React.MouseEvent<HTMLTableRowElement>, item: T) => void;
    onRowContextMenu?: (event: React.MouseEvent<HTMLTableRowElement>, item: T) => void;
}

class DataTable<T> extends React.Component<DataTableProps<T> & WithStyles<CssRules>> {
    render() {
        const { items, columns, classes, onRowClick, onRowContextMenu } = this.props;
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
                                    onClick={event => onRowClick && onRowClick(event, item)}
                                    onContextMenu={event => onRowContextMenu && onRowContextMenu(event, item)}>
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
                </Table> : <Typography
                    className={classes.noItemsInfo}
                    variant="body2"
                    gutterBottom>
                    No items
                </Typography>}
        </div>;
    }
}

type CssRules = "tableBody" | "tableContainer" | "noItemsInfo";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    tableContainer: {
        overflowX: 'auto'
    },
    tableBody: {
        background: theme.palette.background.paper
    },
    noItemsInfo: {
        textAlign: "center",
        padding: theme.spacing.unit
    }
});

export default withStyles(styles)(DataTable);
