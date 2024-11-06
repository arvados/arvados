// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback, CustomTheme, ArvadosTheme } from 'common/custom-theme';
import {
    Table,
    TableBody,
    TableRow,
    TableCell,
    TableHead,
    TableSortLabel,
    IconButton,
    Tooltip,
} from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import classnames from "classnames";
import { DataColumn, DataColumns, SortDirection } from "./data-column";
import { DataTableDefaultView } from "../data-table-default-view/data-table-default-view";
import { DataTableFilters } from "../data-table-filters/data-table-filters";
import { DataTableMultiselectPopover, DataTableMultiselectOption } from "components/data-table-multiselect-popover/data-table-multiselect-popover";
import { DataTableFiltersPopover } from "../data-table-filters/data-table-filters-popover";
import { countNodes, getTreeDirty, createTree } from "models/tree";
import { IconType, PendingIcon } from "components/icon/icon";
import { SvgIconProps } from "@mui/material/SvgIcon";
import ArrowDownwardIcon from "@mui/icons-material/ArrowDownward";
import { isExactlyOneSelected } from "store/multiselect/multiselect-actions";

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
    toggleMSToolbar: (isVisible: boolean) => void;
    setCheckedListOnStore: (checkedList: TCheckedList) => void;
    currentRoute?: string;
    currentRouteUuid: string;
    checkedList: TCheckedList;
    selectedResourceUuid: string;
    setSelectedUuid: (uuid: string | null) => void;
    isNotFound?: boolean;
    detailsPanelResourceUuid?: string;
    loadDetailsPanel: (uuid: string) => void;
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
    | "firstTableHead"
    | "tableHead"
    | "selected"
    | "hovered"
    | "arrow"
    | "arrowButton"
    | "tableCellWorkflows";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: "100%",
    },
    content: {
        display: "inline-block",
        width: "100%",
    },
    tableBody: {
        background: theme.palette.background.paper,
        overflow: "auto",
    },
    noItemsInfo: {
        textAlign: "center",
        padding: theme.spacing(1),
    },
    checkBoxHead: {
        padding: "0",
        display: "flex",
        width: '2rem',
        height: "1.5rem",
        paddingLeft: '0.9rem',
        marginRight: '0.5rem',
        backgroundColor: theme.palette.background.paper,
    },
    checkBoxCell: {
        padding: "0",
        backgroundColor: theme.palette.background.paper,
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
    },
    firstTableCell: {
        paddingLeft: "5px",
    },
    firstTableHead: {
        paddingLeft: "5px",
    },
    tableHead: {
        wordWrap: "break-word",
        paddingRight: "24px",
        color: "#737373",
        fontSize: "0.8125rem",
        backgroundColor: theme.palette.background.paper,
    },
    selected: {
        backgroundColor: `${CustomTheme.palette.grey['300']} !important`
    },
    hovered: {
        backgroundColor: `${CustomTheme.palette.grey['100']} !important`
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
    hoveredIndex: number | null;
};

type DataTableProps<T> = DataTableDataProps<T> & WithStyles<CssRules>;

export const DataTable = withStyles(styles)(
    class Component<T> extends React.Component<DataTableProps<T>> {
        state: DataTableState = {
            isSelected: false,
            isLoaded: false,
            hoveredIndex: null,
        };

        componentDidMount(): void {
            this.initializeCheckedList([]);
            if(((this.props.items.length > 0) && !this.state.isLoaded) || !this.props.working) {
                this.setState({ isLoaded: true });
            }
            if(this.props.detailsPanelResourceUuid !== this.props.selectedResourceUuid) {
                this.props.loadDetailsPanel(this.props.selectedResourceUuid);
            }
        }

        componentDidUpdate(prevProps: Readonly<DataTableProps<T>>, prevState: DataTableState) {
            const { items, currentRouteUuid, setCheckedListOnStore } = this.props;
            const { isSelected } = this.state;
            const singleSelected = isExactlyOneSelected(this.props.checkedList);
            if (prevProps.items !== items) {
                if (isSelected === true) this.setState({ isSelected: false });
                if (items.length) this.initializeCheckedList(items.map((item: any) => item.uuid));
                else setCheckedListOnStore({});
            }
            if (prevProps.currentRoute !== this.props.currentRoute) {
                this.initializeCheckedList([]);
            }
            if (singleSelected && singleSelected !== isExactlyOneSelected(prevProps.checkedList)) {
                this.props.setSelectedUuid(singleSelected);
            }
            if (!singleSelected && !!currentRouteUuid && !this.isAnySelected()) {
                this.props.setSelectedUuid(currentRouteUuid);
            }
            if (!singleSelected && this.isAnySelected()) {
                this.props.setSelectedUuid(null);
            }
            if(prevProps.working === false && this.props.working === true) {
                this.setState({ isLoaded: false });
            }
            if(prevProps.working === true && this.props.working === false) {
                this.setState({ isLoaded: true });
            }
            if((this.props.items.length > 0) && !this.state.isLoaded) {
                this.setState({ isLoaded: true });
            }
        }

        componentWillUnmount(): void {
            this.initializeCheckedList([]);
        }

        checkBoxColumn: DataColumn<any, any> = {
            name: "checkBoxColumn",
            selected: true,
            configurable: false,
            filters: createTree(),
            render: ({uuid}) => {
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

            if(Object.keys(newCheckedList).length === 0){
                for(const uuid of uuids){
                    newCheckedList[uuid] = false
                }
            }

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
            if (!checkedList) return false;
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
            const { items, classes, columns, isNotFound } = this.props;
            const { isLoaded } = this.state;
            if (columns.length && columns[0].name === this.checkBoxColumn.name) columns.shift();
            columns.unshift(this.checkBoxColumn);
            return (
                <div className={classes.root}>
                    <div className={classes.content}>
                        <Table data-cy="data-table" stickyHeader>
                            <TableHead>
                                <TableRow>{this.mapVisibleColumns(this.renderHeadCell)}</TableRow>
                            </TableHead>
                            <TableBody className={classes.tableBody}>{(isLoaded && !isNotFound) && items.map(this.renderBodyRow)}</TableBody>
                        </Table>
                        {(!isLoaded || isNotFound || items.length === 0) && this.renderNoItemsPlaceholder(this.props.columns)}
                    </div>
                </div>
            );
        }

        renderNoItemsPlaceholder = (columns: DataColumns<T, any>) => {
            const { isLoaded } = this.state;
            const { working, isNotFound } = this.props;
            const dirty = columns.some(column => getTreeDirty("")(column.filters));
            if (isNotFound && isLoaded) {
                return (
                    <DataTableDefaultView
                        icon={this.props.defaultViewIcon}
                        messages={["No items found"]}
                    />
                );
            } else
            if (isLoaded === false || working === true) {
                return (
                    <DataTableDefaultView
                        icon={PendingIcon}
                        messages={["Loading data, please wait"]}
                    />
                );
            } else {
                // isLoaded && !working && !isNotFound
                return (
                    <DataTableDefaultView
                        data-cy="data-table-default-view"
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
                        <Tooltip title={this.state.isSelected ? "Deselect all" : "Select all"}>
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
                    className={classnames(classes.tableHead, index === 1 ? classes.firstTableHead : '')}
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
                data-cy="sort-button"
                component="span"
                className={this.props.classes.arrowButton}
                tabIndex={-1}
                size="large">
                <ArrowDownwardIcon
                    {...props}
                    className={classnames(className, this.props.classes.arrow)}
                />
            </IconButton>
        );

        renderBodyRow = (item: any, index: number) => {
            const { onRowClick, onRowDoubleClick, extractKey, classes, selectedResourceUuid, currentRoute } = this.props;
            const { hoveredIndex } = this.state;
            const isRowSelected = item === selectedResourceUuid;
            const getClassnames = (colIndex: number) => {
                if(currentRoute === '/workflows') return classes.tableCellWorkflows;
                if(colIndex === 0) return classnames(classes.checkBoxCell, isRowSelected ? classes.selected : index === hoveredIndex ? classes.hovered : "");
                if(colIndex === 1) return classnames(classes.tableCell, classes.firstTableCell, isRowSelected ? classes.selected : "");
                return classnames(classes.tableCell, isRowSelected ? classes.selected : "");
            };
            const handleHover = (index: number | null) => {
                this.setState({ hoveredIndex: index });
            }

            return (
                <TableRow
                    data-cy={'data-table-row'}
                    hover
                    key={extractKey ? extractKey(item) : index}
                    onClick={event => onRowClick && onRowClick(event, item)}
                    onContextMenu={this.handleRowContextMenu(item)}
                    onDoubleClick={event => onRowDoubleClick && onRowDoubleClick(event, item)}
                    selected={isRowSelected}
                    className={isRowSelected ? classes.selected : ""}
                    onMouseEnter={()=>handleHover(index)}
                    onMouseLeave={()=>handleHover(null)}
                >
                    {this.mapVisibleColumns((column, colIndex) => (
                        <TableCell
                            key={column.key || colIndex}
                            data-cy={column.key || colIndex}
                            className={getClassnames(colIndex)}>
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
