// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
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
} from "@material-ui/core";
import classnames from "classnames";
import { DataColumn, SortDirection } from "./data-column";
import { DataTableDefaultView } from "../data-table-default-view/data-table-default-view";
import { DataTableFilters } from "../data-table-filters/data-table-filters-tree";
import { DataTableMultiselectPopover } from "../data-table-multiselect-popover/data-table-multiselect-popover";
import { DataTableFiltersPopover } from "../data-table-filters/data-table-filters-popover";
import { countNodes, getTreeDirty } from "models/tree";
import { IconType } from "components/icon/icon";
import { SvgIconProps } from "@material-ui/core/SvgIcon";
import ArrowDownwardIcon from "@material-ui/icons/ArrowDownward";
import { createTree } from "models/tree";
import { DataTableMultiselectOption } from "../data-table-multiselect-popover/data-table-multiselect-popover";

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
    toggleMSToolbar: (isVisible: boolean) => void;
    setCheckedListOnStore: (checkedList: TCheckedList) => void;
    checkedList: TCheckedList;
    is404?: boolean;
}

type CssRules =
    | "tableBody"
    | "root"
    | "content"
    | "noItemsInfo"
    | "checkBoxHead"
    | "checkBoxCell"
    | "clickBox"
    | "checkBox"
    | "firstTableCell"
    | "tableCell"
    | "arrow"
    | "arrowButton"
    | "tableCellWorkflows";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        width: "100%",
    },
    content: {
        display: "inline-block",
        width: "100%",
    },
    tableBody: {
        background: theme.palette.background.paper,
    },
    noItemsInfo: {
        textAlign: "center",
        padding: theme.spacing.unit,
    },
    checkBoxHead: {
        padding: "0",
        display: "flex",
        width: '2rem',
        height: "1.5rem",
        paddingLeft: '0.9rem',
        marginRight: '0.5rem'
    },
    checkBoxCell: {
        padding: "0",
    },
    clickBox: {
        display: 'flex',
        width: '1.6rem',
        height: "1.5rem",
        paddingLeft: '0.35rem',
        paddingTop: '0.1rem',
        marginLeft: '0.5rem',
        cursor: "pointer",
    },
    checkBox: {
        cursor: "pointer",
    },
    tableCell: {
        wordWrap: "break-word",
        paddingRight: "24px",
        color: "#737373",
    },
    firstTableCell: {
        paddingLeft: "5px",
    },
    tableCellWorkflows: {
        "&:nth-last-child(2)": {
            padding: "0px",
            maxWidth: "48px",
        },
        "&:last-child": {
            padding: "0px",
            paddingRight: "24px",
            width: "48px",
        },
    },
    arrow: {
        margin: 0,
    },
    arrowButton: {
        color: theme.palette.text.primary,
    },
});

export type TCheckedList = Record<string, boolean>;

type DataTableState = {
    isSelected: boolean;
    isLoaded: boolean;
};

type DataTableProps<T> = DataTableDataProps<T> & WithStyles<CssRules>;

export const DataTable = withStyles(styles)(
    class Component<T> extends React.Component<DataTableProps<T>> {
        state: DataTableState = {
            isSelected: false,
            isLoaded: false,
        };

        componentDidMount(): void {
            this.initializeCheckedList([]);
        }

        componentDidUpdate(prevProps: Readonly<DataTableProps<T>>, prevState: DataTableState) {
            const { items, setCheckedListOnStore } = this.props;
            const { isSelected } = this.state;
            if (prevProps.items !== items) {
                if (isSelected === true) this.setState({ isSelected: false });
                if (items.length) this.initializeCheckedList(items);
                else setCheckedListOnStore({});
            }
            if (prevProps.currentRoute !== this.props.currentRoute) {
                this.initializeCheckedList([])
            }
            if(prevProps.working === true && this.props.working === false) {
                this.setState({ isLoaded: true });
            }
        }

        componentWillUnmount(): void {
            this.initializeCheckedList([])
        }

        checkBoxColumn: DataColumn<any, any> = {
            name: "checkBoxColumn",
            selected: true,
            configurable: false,
            filters: createTree(),
            render: uuid => {
                const { classes, checkedList } = this.props;
                return (
                    <div
                        className={classes.clickBox}
                        onClick={(ev) => {
                            ev.stopPropagation()
                            this.handleSelectOne(uuid)
                        }}
                        onDoubleClick={(ev) => ev.stopPropagation()}
                    >
                        <input
                            data-cy={`multiselect-checkbox-${uuid}`}
                            type='checkbox'
                            name={uuid}
                            className={classes.checkBox}
                            checked={checkedList && checkedList[uuid] ? checkedList[uuid] : false}
                            onChange={() => this.handleSelectOne(uuid)}
                            onDoubleClick={(ev) => ev.stopPropagation()}
                        ></input>
                    </div>
                );
            },
        };

        multiselectOptions: DataTableMultiselectOption[] = [
            { name: "All", fn: list => this.handleSelectAll(list) },
            { name: "None", fn: list => this.handleSelectNone(list) },
            { name: "Invert", fn: list => this.handleInvertSelect(list) },
        ];

        initializeCheckedList = (uuids: any[]): void => {
            const newCheckedList = { ...this.props.checkedList };

            uuids.forEach(uuid => {
                if (!newCheckedList.hasOwnProperty(uuid)) {
                    newCheckedList[uuid] = false;
                }
            });
            for (const key in newCheckedList) {
                if (!uuids.includes(key)) {
                    delete newCheckedList[key];
                }
            }
            this.props.setCheckedListOnStore(newCheckedList);
        };

        isAllSelected = (list: TCheckedList): boolean => {
            for (const key in list) {
                if (list[key] === false) return false;
            }
            return true;
        };

        isAnySelected = (): boolean => {
            const { checkedList } = this.props;
            if (!Object.keys(checkedList).length) return false;
            for (const key in checkedList) {
                if (checkedList[key] === true) return true;
            }
            return false;
        };

        handleSelectOne = (uuid: string): void => {
            const { checkedList } = this.props;
            const newCheckedList = { ...checkedList };
            newCheckedList[uuid] = !checkedList[uuid];
            this.setState({ isSelected: this.isAllSelected(newCheckedList) });
            this.props.setCheckedListOnStore(newCheckedList);
        };

        handleSelectorSelect = (): void => {
            const { checkedList } = this.props;
            const { isSelected } = this.state;
            isSelected ? this.handleSelectNone(checkedList) : this.handleSelectAll(checkedList);
        };

        handleSelectAll = (list: TCheckedList): void => {
            if (Object.keys(list).length) {
                const newCheckedList = { ...list };
                for (const key in newCheckedList) {
                    newCheckedList[key] = true;
                }
                this.setState({ isSelected: true });
                this.props.setCheckedListOnStore(newCheckedList);
            }
        };

        handleSelectNone = (list: TCheckedList): void => {
            const newCheckedList = { ...list };
            for (const key in newCheckedList) {
                newCheckedList[key] = false;
            }
            this.setState({ isSelected: false });
            this.props.setCheckedListOnStore(newCheckedList);
        };

        handleInvertSelect = (list: TCheckedList): void => {
            if (Object.keys(list).length) {
                const newCheckedList = { ...list };
                for (const key in newCheckedList) {
                    newCheckedList[key] = !list[key];
                }
                this.setState({ isSelected: this.isAllSelected(newCheckedList) });
                this.props.setCheckedListOnStore(newCheckedList);
            }
        };

        render() {
            const { items, classes, columns, is404 } = this.props;
            const { isLoaded } = this.state;
            if (columns[0].name === this.checkBoxColumn.name) columns.shift();
            columns.unshift(this.checkBoxColumn);
            return (
                <div className={classes.root}>
                    <div className={classes.content}>
                        <Table>
                            <TableHead>
                                <TableRow>{this.mapVisibleColumns(this.renderHeadCell)}</TableRow>
                            </TableHead>
                            <TableBody className={classes.tableBody}>{(isLoaded && !is404) && items.map(this.renderBodyRow)}</TableBody>
                        </Table>
                        {(!isLoaded || is404 || items.length === 0) && this.renderNoItemsPlaceholder(this.props.columns)}
                    </div>
                </div>
            );
        }

        renderNoItemsPlaceholder = (columns: DataColumns<T, any>) => {
            const dirty = columns.some(column => getTreeDirty("")(column.filters));
            if (this.state.isLoaded === false) {
                return (
                    <DataTableDefaultView 
                        icon={this.props.defaultViewIcon} 
                        messages={["Loading data, please wait"]} 
                    />
                );
            } else if (this.props.is404) {
                return (
                    <DataTableDefaultView 
                        icon={this.props.defaultViewIcon} 
                        messages={["Item not found"]} 
                    />
                );
            } else {
                //if (isLoaded && !is404)
                return (
                    <DataTableDefaultView
                        icon={this.props.defaultViewIcon}
                        messages={this.props.defaultViewMessages}
                        filtersApplied={dirty}
                    />
                );
            }
        };

        renderHeadCell = (column: DataColumn<T, any>, index: number) => {
            const { name, key, renderHeader, filters, sort } = column;
            const { onSortToggle, onFiltersChange, classes, checkedList } = this.props;
            const { isSelected } = this.state;
            return column.name === "checkBoxColumn" ? (
                <TableCell
                    key={key || index}
                    className={classes.checkBoxCell}>
                    <div className={classes.checkBoxHead}>
                        <Tooltip title={this.state.isSelected ? "Deselect All" : "Select All"}>
                            <input
                                type="checkbox"
                                className={classes.checkBox}
                                checked={isSelected}
                                disabled={!this.props.items.length}
                                onChange={this.handleSelectorSelect}></input>
                        </Tooltip>
                        <DataTableMultiselectPopover
                            name={`Options`}
                            disabled={!this.props.items.length}
                            options={this.multiselectOptions}
                            checkedList={checkedList}></DataTableMultiselectPopover>
                    </div>
                </TableCell>
            ) : (
                <TableCell
                    className={index === 1 ? classes.firstTableCell : classes.tableCell}
                    key={key || index}>
                    {renderHeader ? (
                        renderHeader()
                    ) : countNodes(filters) > 0 ? (
                        <DataTableFiltersPopover
                            name={`${name} filters`}
                            mutuallyExclusive={column.mutuallyExclusiveFilters}
                            onChange={filters => onFiltersChange && onFiltersChange(filters, column)}
                            filters={filters}>
                            {name}
                        </DataTableFiltersPopover>
                    ) : sort ? (
                        <TableSortLabel
                            active={sort.direction !== SortDirection.NONE}
                            direction={sort.direction !== SortDirection.NONE ? sort.direction : undefined}
                            IconComponent={this.ArrowIcon}
                            hideSortIcon
                            onClick={() => onSortToggle && onSortToggle(column)}>
                            {name}
                        </TableSortLabel>
                    ) : (
                        <span>{name}</span>
                    )}
                </TableCell>
            );
        };

        ArrowIcon = ({ className, ...props }: SvgIconProps) => (
            <IconButton
                component="span"
                className={this.props.classes.arrowButton}
                tabIndex={-1}>
                <ArrowDownwardIcon
                    {...props}
                    className={classnames(className, this.props.classes.arrow)}
                />
            </IconButton>
        );

        renderBodyRow = (item: any, index: number) => {
            const { onRowClick, onRowDoubleClick, extractKey, classes, currentItemUuid, currentRoute } = this.props;
            return (
                <TableRow
                    data-cy={'data-table-row'}
                    hover
                    key={extractKey ? extractKey(item) : index}
                    onClick={event => onRowClick && onRowClick(event, item)}
                    onContextMenu={this.handleRowContextMenu(item)}
                    onDoubleClick={event => onRowDoubleClick && onRowDoubleClick(event, item)}
                    selected={item === currentItemUuid}>
                    {this.mapVisibleColumns((column, index) => (
                        <TableCell
                            key={column.key || index}
                            className={
                                currentRoute === "/workflows"
                                    ? classes.tableCellWorkflows
                                    : index === 0
                                    ? classes.checkBoxCell
                                    : `${classes.tableCell} ${index === 1 ? classes.firstTableCell : ""}`
                            }>
                            {column.render(item)}
                        </TableCell>
                    ))}
                </TableRow>
            );
        };

        mapVisibleColumns = (fn: (column: DataColumn<T, any>, index: number) => React.ReactElement<any>) => {
            return this.props.columns.filter(column => column.selected).map(fn);
        };

        handleRowContextMenu = (item: T) => (event: React.MouseEvent<HTMLElement>) => this.props.onContextMenu(event, item);
    }
);
