// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    Table,
    TableBody,
    TableRow,
    TableCell,
    TableHead,
    TableSortLabel,
    StyleRulesCallback,
    Theme,
    WithStyles,
    withStyles,
    IconButton,
    Tooltip,
} from '@material-ui/core';
import classnames from 'classnames';
import { DataColumn, SortDirection } from './data-column';
import { DataTableDefaultView } from '../data-table-default-view/data-table-default-view';
import { DataTableFilters } from '../data-table-filters/data-table-filters-tree';
import { DataTableMultiselectPopover } from '../data-table-multiselect-popover/data-table-multiselect-popover';
import { DataTableFiltersPopover } from '../data-table-filters/data-table-filters-popover';
import { countNodes, getTreeDirty } from 'models/tree';
import { IconType, PendingIcon } from 'components/icon/icon';
import { SvgIconProps } from '@material-ui/core/SvgIcon';
import ArrowDownwardIcon from '@material-ui/icons/ArrowDownward';
import { createTree } from 'models/tree';
import arraysAreCongruent from 'validators/arrays-are-congruent';
import { DataTableMultiselectOption } from '../data-table-multiselect-popover/data-table-multiselect-popover';

export type DataColumns<I, R> = Array<DataColumn<I, R>>;

export enum DataTableFetchMode {
    PAGINATED,
    INFINITE,
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

type CssRules =
    | 'tableBody'
    | 'root'
    | 'content'
    | 'noItemsInfo'
    | 'checkBoxHead'
    | 'checkBoxCell'
    | 'checkBox'
    | 'tableCell'
    | 'arrow'
    | 'arrowButton'
    | 'tableCellWorkflows'
    | 'loader';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        width: '100%',
    },
    content: {
        display: 'inline-block',
        width: '100%',
    },
    tableBody: {
        background: theme.palette.background.paper,
    },
    loader: {
        left: '50%',
        marginLeft: '-84px',
        position: 'absolute',
    },
    noItemsInfo: {
        textAlign: 'center',
        padding: theme.spacing.unit,
    },
    checkBoxHead: {
        padding: '0',
        display: 'flex',
    },
    checkBoxCell: {
        padding: '0',
        paddingLeft: '10px',
    },
    checkBox: {
        cursor: 'pointer',
    },
    tableCell: {
        wordWrap: 'break-word',
        paddingRight: '24px',
        color: '#737373',
    },
    tableCellWorkflows: {
        '&:nth-last-child(2)': {
            padding: '0px',
            maxWidth: '48px',
        },
        '&:last-child': {
            padding: '0px',
            paddingRight: '24px',
            width: '48px',
        },
    },
    arrow: {
        margin: 0,
    },
    arrowButton: {
        color: theme.palette.text.primary,
    },
});

type DataTableState = {
    isSelected: boolean;
    checkedList: Record<string, boolean>;
};

const multiselectOptions: DataTableMultiselectOption[] = [
    { name: 'First Option', fn: (checkedList) => console.log('one', checkedList) },
    { name: 'Second Option', fn: (checkedList) => console.log('two', checkedList) },
    { name: 'Third Option', fn: (checkedList) => console.log('three', checkedList) },
];

type DataTableProps<T> = DataTableDataProps<T> & WithStyles<CssRules>;

export const DataTable = withStyles(styles)(
    class Component<T> extends React.Component<DataTableProps<T>> {
        state: DataTableState = {
            isSelected: false,
            checkedList: {},
        };

        componentDidMount(): void {
            this.initializeCheckedList(this.props.items);
        }

        componentDidUpdate(prevProps: Readonly<DataTableProps<T>>, prevState: DataTableState) {
            const { items } = this.props;
            const { checkedList } = this.state;
            if (!arraysAreCongruent(prevProps.items, items)) {
                this.state.isSelected = false;
                this.initializeCheckedList(items);
            }
            if (prevState.checkedList !== checkedList) {
                window.localStorage.setItem('selectedRows', JSON.stringify(checkedList));
            }
        }

        checkBoxColumn: DataColumn<any, any> = {
            name: 'checkBoxColumn',
            selected: true,
            configurable: false,
            filters: createTree(),
            render: (uuid) => (
                <input
                    type='checkbox'
                    name={uuid}
                    className={this.props.classes.checkBox}
                    checked={this.state.checkedList[uuid] ?? false}
                    onChange={() => this.handleCheck(uuid)}
                    onDoubleClick={(ev) => ev.stopPropagation()}
                ></input>
            ),
        };

        initializeCheckedList = (uuids: any[]): void => {
            const { checkedList } = this.state;
            uuids.forEach((uuid) => {
                if (!checkedList.hasOwnProperty(uuid)) {
                    checkedList[uuid] = false;
                }
            });
            for (const key in checkedList) {
                if (!uuids.includes(key)) {
                    delete checkedList[key];
                }
            }
            window.localStorage.setItem('selectedRows', JSON.stringify(checkedList));
        };

        handleSelectorSelect = (): void => {
            const { isSelected, checkedList } = this.state;
            const newCheckedList = { ...checkedList };
            for (const key in newCheckedList) {
                newCheckedList[key] = !isSelected;
            }
            this.setState({ isSelected: !isSelected, checkedList: newCheckedList });
        };

        handleCheck = (uuid: string): void => {
            const { checkedList } = this.state;
            const newCheckedList = { ...checkedList };
            newCheckedList[uuid] = !checkedList[uuid];
            this.setState({ checkedList: newCheckedList });
        };

        handleInvertSelect = (): void => {
            const { checkedList } = this.state;
            const newCheckedList = { ...checkedList };
            for (const key in newCheckedList) {
                newCheckedList[key] = !checkedList[key];
            }
            this.setState({ checkedList: newCheckedList });
        };

        render() {
            const { items, classes, working, columns } = this.props;
            if (columns[0].name === this.checkBoxColumn.name) columns.shift();
            columns.unshift(this.checkBoxColumn);
            return (
                <div className={classes.root}>
                    <div className={classes.content}>
                        <Table>
                            <TableHead>
                                <TableRow>{this.mapVisibleColumns(this.renderHeadCell)}</TableRow>
                            </TableHead>
                            <TableBody className={classes.tableBody}>{!working && items.map(this.renderBodyRow)}</TableBody>
                        </Table>
                        {!!working && (
                            <div className={classes.loader}>
                                <DataTableDefaultView icon={PendingIcon} messages={['Loading data, please wait.']} />
                            </div>
                        )}
                        {items.length === 0 && !working && this.renderNoItemsPlaceholder(this.props.columns)}
                    </div>
                </div>
            );
        }

        renderNoItemsPlaceholder = (columns: DataColumns<T, any>) => {
            const dirty = columns.some((column) => getTreeDirty('')(column.filters));
            return <DataTableDefaultView icon={this.props.defaultViewIcon} messages={this.props.defaultViewMessages} filtersApplied={dirty} />;
        };

        renderHeadCell = (column: DataColumn<T, any>, index: number) => {
            const { name, key, renderHeader, filters, sort } = column;
            const { onSortToggle, onFiltersChange, classes } = this.props;
            return column.name === 'checkBoxColumn' ? (
                <TableCell key={key || index} className={classes.checkBoxCell}>
                    <div className={classes.checkBoxHead}>
                        <Tooltip title={this.state.isSelected ? 'Deselect All' : 'Select All'}>
                            <input type='checkbox' className={classes.checkBox} checked={this.state.isSelected} onChange={this.handleSelectorSelect}></input>
                        </Tooltip>
                        <DataTableMultiselectPopover name={`Options`} options={multiselectOptions} checkedList={this.state.checkedList}></DataTableMultiselectPopover>
                    </div>
                </TableCell>
            ) : (
                <TableCell className={classes.tableCell} key={key || index}>
                    {renderHeader ? (
                        renderHeader()
                    ) : countNodes(filters) > 0 ? (
                        <DataTableFiltersPopover
                            name={`${name} filters`}
                            mutuallyExclusive={column.mutuallyExclusiveFilters}
                            onChange={(filters) => onFiltersChange && onFiltersChange(filters, column)}
                            filters={filters}
                        >
                            {name}
                        </DataTableFiltersPopover>
                    ) : sort ? (
                        <TableSortLabel
                            active={sort.direction !== SortDirection.NONE}
                            direction={sort.direction !== SortDirection.NONE ? sort.direction : undefined}
                            IconComponent={this.ArrowIcon}
                            hideSortIcon
                            onClick={() => onSortToggle && onSortToggle(column)}
                        >
                            {name}
                        </TableSortLabel>
                    ) : (
                        <span>{name}</span>
                    )}
                </TableCell>
            );
        };

        ArrowIcon = ({ className, ...props }: SvgIconProps) => (
            <IconButton component='span' className={this.props.classes.arrowButton} tabIndex={-1}>
                <ArrowDownwardIcon {...props} className={classnames(className, this.props.classes.arrow)} />
            </IconButton>
        );

        renderBodyRow = (item: any, index: number) => {
            const { onRowClick, onRowDoubleClick, extractKey, classes, currentItemUuid, currentRoute } = this.props;
            return (
                <TableRow
                    hover
                    key={extractKey ? extractKey(item) : index}
                    onClick={(event) => onRowClick && onRowClick(event, item)}
                    onContextMenu={this.handleRowContextMenu(item)}
                    onDoubleClick={(event) => onRowDoubleClick && onRowDoubleClick(event, item)}
                    selected={item === currentItemUuid}
                >
                    {this.mapVisibleColumns((column, index) => (
                        <TableCell
                            key={column.key || index}
                            className={currentRoute === '/workflows' ? classes.tableCellWorkflows : index === 0 ? classes.checkBoxCell : classes.tableCell}
                        >
                            {column.render(item)}
                        </TableCell>
                    ))}
                </TableRow>
            );
        };

        mapVisibleColumns = (fn: (column: DataColumn<T, any>, index: number) => React.ReactElement<any>) => {
            return this.props.columns.filter((column) => column.selected).map(fn);
        };

        handleRowContextMenu = (item: T) => (event: React.MouseEvent<HTMLElement>) => this.props.onContextMenu(event, item);
    }
);
