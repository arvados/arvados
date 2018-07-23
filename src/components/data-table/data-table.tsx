// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Table, TableBody, TableRow, TableCell, TableHead, TableSortLabel, StyleRulesCallback, Theme, WithStyles, withStyles, Typography } from '@material-ui/core';
import { DataColumn, SortDirection } from './data-column';
import { DataTableFilters,  DataTableFilterItem } from "../data-table-filters/data-table-filters";

export type DataColumns<T, F extends DataTableFilterItem = DataTableFilterItem> = Array<DataColumn<T, F>>;

export interface DataTableDataProps<T> {
    items: T[];
    columns: DataColumns<T>;
    onRowClick: (event: React.MouseEvent<HTMLTableRowElement>, item: T) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: T) => void;
    onRowDoubleClick: (event: React.MouseEvent<HTMLTableRowElement>, item: T) => void;
    onSortToggle: (column: DataColumn<T>) => void;
    onFiltersChange: (filters: DataTableFilterItem[], column: DataColumn<T>) => void;
    extractKey?: (item: T) => React.Key;
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

type DataTableProps<T> = DataTableDataProps<T> & WithStyles<CssRules>;

export const DataTable = withStyles(styles)(
    class Component<T> extends React.Component<DataTableProps<T>> {
        render() {
            const { items, classes } = this.props;
            return <div
                className={classes.tableContainer}>
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
            return <TableCell key={key || index} style={{ width: column.width, minWidth: column.width }}>
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
                                active={sortDirection !== SortDirection.None}
                                direction={sortDirection !== SortDirection.None ? sortDirection : undefined}
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
            const { onRowClick, onRowDoubleClick, extractKey } = this.props;
            return <TableRow
                hover
                key={extractKey ? extractKey(item) : index}
                onClick={event => onRowClick && onRowClick(event, item)}
                onContextMenu={this.handleRowContextMenu(item)}
                onDoubleClick={event => onRowDoubleClick && onRowDoubleClick(event, item) }>
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

        handleRowContextMenu = (item: T) =>
            (event: React.MouseEvent<HTMLElement>) =>
                this.props.onContextMenu(event, item)

    }
);
