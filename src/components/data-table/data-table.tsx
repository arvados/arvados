// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Table, TableBody, TableRow, TableCell, TableHead, TableSortLabel, StyleRulesCallback, Theme, WithStyles, withStyles, Typography } from '@material-ui/core';
import { DataColumn, SortDirection } from './data-column';
import DataTableFilters, { DataTableFilterItem } from "../data-table-filters/data-table-filters";

export type DataColumns<T, F extends DataTableFilterItem = DataTableFilterItem> = Array<DataColumn<T, F>>;
export interface DataItem {
    key: React.Key;
}
export interface DataTableProps<T> {
    items: T[];
    columns: DataColumns<T>;
    onRowClick: (event: React.MouseEvent<HTMLTableRowElement>, item: T) => void;
    onRowContextMenu: (event: React.MouseEvent<HTMLTableRowElement>, item: T) => void;
    onSortToggle: (column: DataColumn<T>) => void;
    onFiltersChange: (filters: DataTableFilterItem[], column: DataColumn<T>) => void;
}

class DataTable<T extends DataItem> extends React.Component<DataTableProps<T> & WithStyles<CssRules>> {
    render() {
        const { items, classes } = this.props;
        return <div className={classes.tableContainer}>
            <Table>
                <TableHead>
                    <TableRow>
                        {this.mapVisibleColumns(this.renderHeadCell)}
                    </TableRow>
                </TableHead>
                <TableBody className={classes.tableBody}>
                    {items.map(this.renderBodyRow)}
                </TableBody>
            </Table>
        </div>;
    }

    renderHeadCell = (column: DataColumn<T>, index: number) => {
        const { name, key, renderHeader, filters, sortDirection } = column;
        const { onSortToggle, onFiltersChange } = this.props;
        return <TableCell key={key || index} style={{width: column.width, minWidth: column.width}}>
            {renderHeader ?
                renderHeader() :
                filters
                    ? <DataTableFilters
                        name={`${name} filters`}
                        onChange={filters =>
                            onFiltersChange &&
                            onFiltersChange(filters, column)}
                        filters={filters}>
                        {name}
                    </DataTableFilters>
                    : sortDirection
                        ? <TableSortLabel
                            active={sortDirection !== "none"}
                            direction={sortDirection !== "none" ? sortDirection : undefined}
                            onClick={() =>
                                onSortToggle &&
                                onSortToggle(column)}>
                            {name}
                        </TableSortLabel>
                        : <span>
                            {name}
                        </span>}
        </TableCell>;
    }

    renderBodyRow = (item: T, index: number) => {
        const { onRowClick, onRowContextMenu } = this.props;
        return <TableRow
            hover
            key={item.key}
            onClick={event => onRowClick && onRowClick(event, item)}
            onContextMenu={event => onRowContextMenu && onRowContextMenu(event, item)}>
            {this.mapVisibleColumns((column, index) => (
                <TableCell key={column.key || index}>
                    {column.render(item)}
                </TableCell>
            ))}
        </TableRow>;
    }

    mapVisibleColumns = (fn: (column: DataColumn<T>, index: number) => React.ReactElement<any>) => {
        return this.props.columns.filter(column => column.selected).map(fn);
    }

}

type CssRules = "tableBody" | "tableContainer" | "noItemsInfo";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    tableContainer: {
        overflowX: 'auto',
        overflowY: 'hidden'
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
