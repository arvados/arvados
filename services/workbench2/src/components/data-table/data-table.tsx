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
import { IconType } from "components/icon/icon";
import { SvgIconProps } from "@mui/material/SvgIcon";
import ArrowDownwardIcon from "@mui/icons-material/ArrowDownward";
import { isExactlyOneSelected } from "store/multiselect/multiselect-actions";
import { LoadingIndicator } from "components/loading-indicator/loading-indicator";
import { ColumnFilterCounts } from "components/data-table-filters/data-table-filters-tree";

export enum DataTableFetchMode {
    PAGINATED,
    INFINITE,
}

const LOADING_PLACEHOLDER_COUNT = 3;

enum DataTableContentType {
    ROWS,
    NOTFOUND,
    LOADING,
    EMPTY,
};

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
    columnFilterCounts: ColumnFilterCounts;
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
    | "tableCellWorkflows"
    | "loadingRow"
    | "hiddenCell"
    | "skeleton";

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
        cursor: "pointer",
    },
    clickBox: {
        display: 'flex',
        width: '1.6rem',
        height: "1.5rem",
        paddingLeft: '0.35rem',
        paddingTop: '0.1rem',
        marginLeft: '0.5rem',
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
    loadingRow: {
        height: "49px",
    },
    hiddenCell: {
        position: "relative",
        "& > *": {
            visibility: "hidden",
        },
    },
    skeleton: {
        visibility: "visible",
        position: "absolute",
        top: 0,
        left: 0,
        width: "100%",
        height: "100%",
        paddingLeft: "5px",
        paddingRight: "24px",
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        gap: "8px",
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

        shouldComponentUpdate( nextProps: Readonly<DataTableProps<T>>, nextState: Readonly<DataTableState>, nextContext: any ): boolean {
            const { items, currentRouteUuid, isNotFound, checkedList, columns, working, columnFilterCounts } = this.props;
            const { isSelected, isLoaded, hoveredIndex } = this.state;
            return items !== nextProps.items
                || currentRouteUuid !== nextProps.currentRouteUuid
                || isNotFound !== nextProps.isNotFound
                || isLoaded !== nextState.isLoaded
                || isSelected !== nextState.isSelected
                || hoveredIndex !== nextState.hoveredIndex
                || checkedList !== nextProps.checkedList
                || columns !== nextProps.columns
                || columnFilterCounts !== nextProps.columnFilterCounts
                || working !== nextProps.working;
        }

        componentDidUpdate(prevProps: Readonly<DataTableProps<T>>, prevState: DataTableState) {
            const { items, currentRouteUuid, checkedList, setCheckedListOnStore } = this.props;
            const { isSelected } = this.state;
            const singleSelected = isExactlyOneSelected(this.props.checkedList);
            if (prevProps.items !== items) {
                if (isSelected === true) this.setState({ isSelected: false });
                if (items.length) this.initializeCheckedList(items);
                else setCheckedListOnStore({});
            }
            if (items.length && checkedList && (Object.keys(checkedList)).length === 0) {
                this.initializeCheckedList(items);
            }
            if (prevProps.currentRoute !== this.props.currentRoute) {
                this.initializeCheckedList([]);
            }
            if (this.state.isLoaded){
                if (singleSelected && singleSelected !== isExactlyOneSelected(prevProps.checkedList)) {
                    this.props.setSelectedUuid(singleSelected);
                }
                if (!singleSelected && !!currentRouteUuid && !this.isAnySelected()) {
                    this.props.setSelectedUuid(currentRouteUuid);
                }
                if (!singleSelected && this.isAnySelected()) {
                    this.props.setSelectedUuid(null);
                }
                if (this.isAnySelected()) {
                    this.setState({ isSelected: true })
                }
            }
            if(prevProps.working === false && this.props.working === true) {
                this.setState({ isLoaded: false });
                this.handleSelectNone(this.props.checkedList);
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
            const newCheckedList = uuids
                .reduce((acc, curr) => ({
                    ...acc,
                    [curr]: false
                }), {} as TCheckedList);
            this.props.setCheckedListOnStore(newCheckedList);
        };

        isAllSelected = (list: TCheckedList): boolean => {
            return Object.keys(list)
                .every((key) => list[key] === true);
        };

        isAnySelected = (): boolean => {
            const { checkedList } = this.props;
            return !!checkedList
                && !!Object.keys(checkedList).length
                && Object.keys(checkedList).some((key) => checkedList[key] === true);
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
                const newCheckedList = Object.keys(list)
                    .reduce((acc, curr) => ({
                        ...acc,
                        [curr]: true
                    }), {} as TCheckedList);
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

        /**
         * Helper to contain display state logic to avoid recalculating in multiple places
         * @param items Data table items array
         * @returns An enum value representing what should be displayed
         */
        getDataTableContentType = (items: T[]): DataTableContentType => {
            const { working, isNotFound } = this.props;
            const { isLoaded } = this.state;

            if (isLoaded && !isNotFound && !!items.length && !working) {
                return DataTableContentType.ROWS;
            } else if (isNotFound && isLoaded) {
                return DataTableContentType.NOTFOUND;
            } else if (isLoaded === false || working === true) {
                return DataTableContentType.LOADING;
            } else {
                // isLoaded && !working && !isNotFound
                return DataTableContentType.EMPTY;
            }
        };

        render() {
            const { items, classes, columns } = this.props;
            const dataTableContentType = this.getDataTableContentType(items);
            if (columns.length && columns[0].name === this.checkBoxColumn.name) columns.shift();
            columns.unshift(this.checkBoxColumn);
            return (
                <div className={classes.root}>
                    <div className={classes.content}>
                        <Table data-cy="data-table" stickyHeader>
                            <TableHead>
                                <TableRow>{this.mapVisibleColumns(this.renderHeadCell)}</TableRow>
                            </TableHead>
                            <TableBody className={classes.tableBody}>
                                {this.renderBody(items, dataTableContentType)}
                            </TableBody>
                        </Table>
                        {this.renderNoItemsPlaceholder(dataTableContentType, this.props.columns)}
                    </div>
                </div>
            );
        }

        renderLoadingPlaceholder = () => {
            return <>
                {(new Array(LOADING_PLACEHOLDER_COUNT).fill(0)).map(() => {
                    return <TableRow hover className={this.props.classes.loadingRow}>
                        {this.mapVisibleColumns((column, colIndex) => (
                            <TableCell
                                key={column.key || colIndex}
                                data-cy={column.key || colIndex}
                                >
                                <LoadingIndicator />
                            </TableCell>
                        ))}
                    </TableRow>
                })}
            </>;
        };

        renderNoItemsPlaceholder = (dataTableContentType: DataTableContentType, columns: DataColumns<T, any>) => {
            const dirty = columns.some(column => getTreeDirty("")(column.filters));
            if (dataTableContentType === DataTableContentType.NOTFOUND) {
                return (
                    <DataTableDefaultView
                        icon={this.props.defaultViewIcon}
                        messages={["No items found"]}
                    />
                );
            } else if (dataTableContentType === DataTableContentType.EMPTY) {
                // isLoaded && !working && !isNotFound
                return (
                    <DataTableDefaultView
                        data-cy="data-table-default-view"
                        icon={this.props.defaultViewIcon}
                        messages={this.props.defaultViewMessages}
                        filtersApplied={dirty}
                    />
                );
            } else {
                return <></>;
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
                            columnFilterCount={this.props.columnFilterCounts?.[name] || {}}
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

        renderBody = (items: any[], dataTableContentType: DataTableContentType) => {
            if (items.length) {
                // Have items, renderBodyRow renders rows or skeleton over rows
                return items.map((item, index) => this.renderBodyRow(item, index, dataTableContentType));
            } else if (dataTableContentType === DataTableContentType.LOADING) {
                // No rows and loading, use static skeleton
                return this.renderLoadingPlaceholder();
            }
            // No rows and not loading, let empty view outside table body display
            return <></>;
        };

        renderBodyRow = (item: any, index: number, dataTableContentType: DataTableContentType) => {
            const { onRowClick, onRowDoubleClick, extractKey, classes, currentRoute, checkedList } = this.props;
            const { hoveredIndex } = this.state;
            const isRowSelected = checkedList && checkedList[item] === true;
            const getCellClassnames = (colIndex: number) => {
                let cellClasses: string[] = [];
                if (dataTableContentType === DataTableContentType.LOADING) cellClasses.push(classes.hiddenCell);
                if(currentRoute === '/workflows') return classnames(cellClasses, classes.tableCellWorkflows);
                if(colIndex === 0) return classnames(cellClasses, classes.checkBoxCell, isRowSelected ? classes.selected : index === hoveredIndex ? classes.hovered : "");
                if(colIndex === 1) return classnames(cellClasses, classes.tableCell, classes.firstTableCell, isRowSelected ? classes.selected : "");
                return classnames(cellClasses, classes.tableCell, isRowSelected ? classes.selected : "");
            };
            const handleHover = (index: number | null) => {
                this.setState({ hoveredIndex: index });
            }

            const noopWhenLoading = (func) => {
                if (dataTableContentType === DataTableContentType.LOADING) {
                    return (e) => e.preventDefault();
                } else {
                    return func;
                }
            }

            return (
                <TableRow
                    data-cy={'data-table-row'}
                    hover
                    key={extractKey ? extractKey(item) : index}
                    onClick={noopWhenLoading(event => onRowClick && onRowClick(event, item))}
                    onContextMenu={noopWhenLoading(this.handleRowContextMenu(item))}
                    onDoubleClick={noopWhenLoading(event => onRowDoubleClick && onRowDoubleClick(event, item))}
                    selected={isRowSelected}
                    className={isRowSelected ? classes.selected : ""}
                    onMouseEnter={()=>handleHover(index)}
                    onMouseLeave={()=>handleHover(null)}
                >
                    {this.mapVisibleColumns((column, colIndex) => (
                        <TableCell
                            key={column.key || colIndex}
                            data-cy={column.key || colIndex}
                            className={getCellClassnames(colIndex)}>
                            {column.render(item)}
                            {dataTableContentType === DataTableContentType.LOADING && <LoadingIndicator inline={true} containerClassName={classes.skeleton} />}
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
