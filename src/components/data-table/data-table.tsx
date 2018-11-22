// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Table, TableBody, TableRow, TableCell, TableHead, TableSortLabel, StyleRulesCallback, Theme, WithStyles, withStyles } from '@material-ui/core';
import { DataColumn, SortDirection } from './data-column';
import { DataTableDefaultView } from '../data-table-default-view/data-table-default-view';
import { DataTableFilters } from '../data-table-filters/data-table-filters-tree';
import { DataTableFiltersPopover } from '../data-table-filters/data-table-filters-popover';
import { countNodes } from '~/models/tree';

export type DataColumns<T> = Array<DataColumn<T>>;

export interface DataTableDataProps<T> {
    items: T[];
    columns: DataColumns<T>;
    onRowClick: (event: React.MouseEvent<HTMLTableRowElement>, item: T) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: T) => void;
    onRowDoubleClick: (event: React.MouseEvent<HTMLTableRowElement>, item: T) => void;
    onSortToggle: (column: DataColumn<T>) => void;
    onFiltersChange: (filters: DataTableFilters, column: DataColumn<T>) => void;
    extractKey?: (item: T) => React.Key;
    working?: boolean;
    defaultView?: React.ReactNode;
}

type CssRules = "tableBody" | "root" | "content" | "noItemsInfo" | 'tableCell';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        overflowX: 'auto',
        overflowY: 'hidden'
    },
    content: {
        display: 'inline-block',
        width: '100%'
    },
    tableBody: {
        background: theme.palette.background.paper
    },
    noItemsInfo: {
        textAlign: "center",
        padding: theme.spacing.unit
    },
    tableCell: {
        wordWrap: 'break-word'
    }
});

type DataTableProps<T> = DataTableDataProps<T> & WithStyles<CssRules>;

export const DataTable = withStyles(styles)(
    class Component<T> extends React.Component<DataTableProps<T>> {
        render() {
            const { items, classes } = this.props;
            return <div className={classes.root}>
                <div className={classes.content}>
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
                    {items.length === 0 && this.props.working !== undefined && !this.props.working && this.renderNoItemsPlaceholder()}
                </div>
            </div>;
        }

        renderNoItemsPlaceholder = () => {
            return this.props.defaultView
                ? this.props.defaultView
                : <DataTableDefaultView/>;
        }

        renderHeadCell = (column: DataColumn<T>, index: number) => {
            const { name, key, renderHeader, filters, sortDirection } = column;
            const { onSortToggle, onFiltersChange } = this.props;
            return <TableCell key={key || index}>
                {renderHeader ?
                    renderHeader() :
                    countNodes(filters) > 0
                        ? <DataTableFiltersPopover
                            name={`${name} filters`}
                            onChange={filters =>
                                onFiltersChange &&
                                onFiltersChange(filters, column)}
                            filters={filters}>
                            {name}
                        </DataTableFiltersPopover>
                        : sortDirection
                            ? <TableSortLabel
                                active={sortDirection !== SortDirection.NONE}
                                direction={sortDirection !== SortDirection.NONE ? sortDirection : undefined}
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
            const { onRowClick, onRowDoubleClick, extractKey, classes } = this.props;
            return <TableRow
                hover
                key={extractKey ? extractKey(item) : index}
                onClick={event => onRowClick && onRowClick(event, item)}
                onContextMenu={this.handleRowContextMenu(item)}
                onDoubleClick={event => onRowDoubleClick && onRowDoubleClick(event, item)}>
                {this.mapVisibleColumns((column, index) => (
                    <TableCell key={column.key || index} className={classes.tableCell}>
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
