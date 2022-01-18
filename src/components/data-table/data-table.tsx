// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Table, TableBody, TableRow, TableCell, TableHead, TableSortLabel, StyleRulesCallback, Theme, WithStyles, withStyles, IconButton } from '@material-ui/core';
import classnames from 'classnames';
import { DataColumn, SortDirection } from './data-column';
import { DataTableDefaultView } from '../data-table-default-view/data-table-default-view';
import { DataTableFilters } from '../data-table-filters/data-table-filters-tree';
import { DataTableFiltersPopover } from '../data-table-filters/data-table-filters-popover';
import { countNodes } from 'models/tree';
import { ProjectIcon } from 'components/icon/icon';
import { SvgIconProps } from '@material-ui/core/SvgIcon';
import ArrowDownwardIcon from '@material-ui/icons/ArrowDownward';

export type DataColumns<T> = Array<DataColumn<T>>;

export enum DataTableFetchMode {
    PAGINATED,
    INFINITE
}

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
    currentItemUuid?: string;
    currentRoute?: string;
}

type CssRules = "tableBody" | "root" | "content" | "noItemsInfo" | 'tableCell' | 'arrow' | 'arrowButton' | 'tableCellWorkflows' | 'loader';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        width: '100%',
    },
    content: {
        display: 'inline-block',
        width: '100%',
    },
    tableBody: {
        background: theme.palette.background.paper
    },
    loader: {
        left: '50%',
        marginLeft: '-84px',
        position: 'absolute'
    },
    noItemsInfo: {
        textAlign: "center",
        padding: theme.spacing.unit
    },
    tableCell: {
        wordWrap: 'break-word',
        paddingRight: '24px'
    },
    tableCellWorkflows: {
        '&:nth-last-child(2)': {
            padding: '0px',
            maxWidth: '48px'
        },
        '&:last-child': {
            padding: '0px',
            paddingRight: '24px',
            width: '48px'
        }
    },
    arrow: {
        margin: 0
    },
    arrowButton: {
        color: theme.palette.text.primary
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
                            {
                                this.props.working ?
                                <div className={classes.loader}>
                                    <DataTableDefaultView
                                        icon={ProjectIcon}
                                        messages={['Loading data, please wait.']} />
                                </div> : items.map(this.renderBodyRow)
                            }
                        </TableBody>
                    </Table>
                    {items.length === 0 && this.props.working !== undefined && !this.props.working && this.renderNoItemsPlaceholder()}
                </div>
            </div>;
        }

        renderNoItemsPlaceholder = () => {
            return this.props.defaultView
                ? this.props.defaultView
                : <DataTableDefaultView />;
        }

        renderHeadCell = (column: DataColumn<T>, index: number) => {
            const { name, key, renderHeader, filters, sortDirection } = column;
            const { onSortToggle, onFiltersChange, classes } = this.props;
            return <TableCell className={classes.tableCell} key={key || index}>
                {renderHeader ?
                    renderHeader() :
                    countNodes(filters) > 0
                        ? <DataTableFiltersPopover
                            name={`${name} filters`}
                            mutuallyExclusive={column.mutuallyExclusiveFilters}
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
                                IconComponent={this.ArrowIcon}
                                hideSortIcon
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

        ArrowIcon = ({ className, ...props }: SvgIconProps) => (
            <IconButton component='span' className={this.props.classes.arrowButton} tabIndex={-1}>
                <ArrowDownwardIcon {...props} className={classnames(className, this.props.classes.arrow)} />
            </IconButton>
        )

        renderBodyRow = (item: any, index: number) => {
            const { onRowClick, onRowDoubleClick, extractKey, classes, currentItemUuid, currentRoute } = this.props;
            return <TableRow
                hover
                key={extractKey ? extractKey(item) : index}
                onClick={event => onRowClick && onRowClick(event, item)}
                onContextMenu={this.handleRowContextMenu(item)}
                onDoubleClick={event => onRowDoubleClick && onRowDoubleClick(event, item)}
                selected={item === currentItemUuid}>
                {this.mapVisibleColumns((column, index) => (
                    <TableCell key={column.key || index} className={currentRoute === '/workflows' ? classes.tableCellWorkflows : classes.tableCell}>
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
