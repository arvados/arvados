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
import { countNodes, getTreeDirty } from 'models/tree';
import { IconType, PendingIcon } from 'components/icon/icon';
import { SvgIconProps } from '@material-ui/core/SvgIcon';
import ArrowDownwardIcon from '@material-ui/icons/ArrowDownward';

export type DataColumns<I, R> = Array<DataColumn<I, R>>;

export enum DataTableFetchMode {
    PAGINATED,
    INFINITE
}

export interface DataTableDataProps<I> {
    items: I[];
    columns: DataColumns<I, any>;
    onRowClick: (event: React.MouseEvent<HTMLTableRowElement>, item: I) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: I) => void;
    onRowDoubleClick: (event: React.MouseEvent<HTMLTableRowElement>, item: I) => void;
    onSortToggle: (column: DataColumn<I, any>) => void;
    onFiltersChange: (filters: DataTableFilters, column: DataColumn<I, any>) => void;
    extractKey?: (item: I) => React.Key;
    working?: boolean;
    defaultViewIcon?: IconType;
    defaultViewMessages?: string[];
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
        paddingRight: '24px',
        color: '#737373'

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
            const { items, classes, working } = this.props;
            return <div className={classes.root}>
                <div className={classes.content}>
                    <Table>
                        <TableHead>
                            <TableRow>
                                {this.mapVisibleColumns(this.renderHeadCell)}
                            </TableRow>
                        </TableHead>
                        <TableBody className={classes.tableBody}>
                            { !working && items.map(this.renderBodyRow) }
                        </TableBody>
                    </Table>
                    { !!working &&
                        <div className={classes.loader}>
                            <DataTableDefaultView
                                icon={PendingIcon}
                                messages={['Loading data, please wait.']} />
                        </div> }
                    {items.length === 0 && !working && this.renderNoItemsPlaceholder(this.props.columns)}
                </div>
            </div>;
        }

        renderNoItemsPlaceholder = (columns: DataColumns<T, any>) => {
            const dirty = columns.some((column) => getTreeDirty('')(column.filters));
            return <DataTableDefaultView
                icon={this.props.defaultViewIcon}
                messages={this.props.defaultViewMessages}
                filtersApplied={dirty} />;
        }

        renderHeadCell = (column: DataColumn<T, any>, index: number) => {
            const { name, key, renderHeader, filters, sort } = column;
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
                        : sort
                            ? <TableSortLabel
                                active={sort.direction !== SortDirection.NONE}
                                direction={sort.direction !== SortDirection.NONE ? sort.direction : undefined}
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
                {this.mapVisibleColumns((column, index) => <TableCell key={column.key || index} className={currentRoute === '/workflows' ? classes.tableCellWorkflows : classes.tableCell}>
                        {column.render(item)}
                    </TableCell>
                )}
            </TableRow>;
        }

        mapVisibleColumns = (fn: (column: DataColumn<T, any>, index: number) => React.ReactElement<any>) => {
            return this.props.columns.filter(column => column.selected).map(fn);
        }

        handleRowContextMenu = (item: T) =>
            (event: React.MouseEvent<HTMLElement>) =>
                this.props.onContextMenu(event, item)

    }
);
