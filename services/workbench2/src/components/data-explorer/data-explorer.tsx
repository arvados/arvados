// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Grid, Paper, Toolbar, StyleRulesCallback, withStyles, WithStyles, TablePagination, IconButton, Tooltip, Button } from "@material-ui/core";
import { ColumnSelector } from "components/column-selector/column-selector";
import { DataTable, DataColumns, DataTableFetchMode } from "components/data-table/data-table";
import { DataColumn } from "components/data-table/data-column";
import { SearchInput } from "components/search-input/search-input";
import { ArvadosTheme } from "common/custom-theme";
import { MultiselectToolbar } from "components/multiselect-toolbar/MultiselectToolbar";
import { TCheckedList } from "components/data-table/data-table";
import { createTree } from "models/tree";
import { DataTableFilters } from "components/data-table-filters/data-table-filters-tree";
import { CloseIcon, IconType, MaximizeIcon, UnMaximizeIcon, MoreVerticalIcon } from "components/icon/icon";
import { PaperProps } from "@material-ui/core/Paper";
import { MPVPanelProps } from "components/multi-panel-view/multi-panel-view";

type CssRules = "titleWrapper" | "msToolbarStyles" | "subpanelToolbarStyles" | "searchBox" | "headerMenu" | "toolbar" | "footer" | "root" | "moreOptionsButton" | "title" | 'subProcessTitle' | "dataTable" | "container";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    titleWrapper: {
        display: "flex",
        justifyContent: "space-between",
    },
    msToolbarStyles: {
        paddingTop: "0.6rem",
    },
    subpanelToolbarStyles: {
        paddingTop: "1.2rem",
    },
    searchBox: {
        paddingBottom: 0,
    },
    toolbar: {
        paddingTop: 0,
        paddingRight: theme.spacing.unit,
        paddingLeft: "10px",
    },
    footer: {
        overflow: "auto",
    },
    root: {
        height: "100%",
        flex: 1,
        overflowY: "auto",
    },
    moreOptionsButton: {
        padding: 0,
    },
    title: {
        display: "inline-block",
        paddingLeft: theme.spacing.unit * 2,
        paddingTop: theme.spacing.unit * 2,
        fontSize: "18px",
        paddingRight: "10px",
    },
    subProcessTitle: {
        display: "inline-block",
        paddingLeft: theme.spacing.unit * 2,
        paddingTop: theme.spacing.unit * 2,
        fontSize: "18px",
        flexGrow: 0,
        paddingRight: "10px",
    },
    dataTable: {
        height: "100%",
        overflow: "auto",
    },
    container: {
        height: "100%",
    },
    headerMenu: {
        marginLeft: "auto",
        flexBasis: "initial",
        flexGrow: 0,
    },
});

interface DataExplorerDataProps<T> {
    fetchMode: DataTableFetchMode;
    items: T[];
    itemsAvailable: number;
    columns: DataColumns<T, any>;
    searchLabel?: string;
    searchValue: string;
    rowsPerPage: number;
    rowsPerPageOptions: number[];
    page: number;
    contextMenuColumn: boolean;
    defaultViewIcon?: IconType;
    defaultViewMessages?: string[];
    working?: boolean;
    currentRoute?: string;
    hideColumnSelector?: boolean;
    paperProps?: PaperProps;
    actions?: React.ReactNode;
    hideSearchInput?: boolean;
    title?: React.ReactNode;
    progressBar?: React.ReactNode;
    paperKey?: string;
    currentRouteUuid: string;
    selectedResourceUuid: string;
    elementPath?: string;
    isMSToolbarVisible: boolean;
    checkedList: TCheckedList;
    isNotFound: boolean;
}

interface DataExplorerActionProps<T> {
    onSetColumns: (columns: DataColumns<T, any>) => void;
    onSearch: (value: string) => void;
    onRowClick: (item: T) => void;
    onRowDoubleClick: (item: T) => void;
    onColumnToggle: (column: DataColumn<T, any>) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: T) => void;
    onSortToggle: (column: DataColumn<T, any>) => void;
    onFiltersChange: (filters: DataTableFilters, column: DataColumn<T, any>) => void;
    onChangePage: (page: number) => void;
    onChangeRowsPerPage: (rowsPerPage: number) => void;
    onLoadMore: (page: number) => void;
    extractKey?: (item: T) => React.Key;
    toggleMSToolbar: (isVisible: boolean) => void;
    setCheckedListOnStore: (checkedList: TCheckedList) => void;
    setSelectedUuid: (uuid: string) => void;
}

type DataExplorerProps<T> = DataExplorerDataProps<T> & DataExplorerActionProps<T> & WithStyles<CssRules> & MPVPanelProps;

export const DataExplorer = withStyles(styles)(
    class DataExplorerGeneric<T> extends React.Component<DataExplorerProps<T>> {
        state = {
            msToolbarInDetailsCard: true,
        };

        multiSelectToolbarInTitle = !this.props.title && !this.props.progressBar;

        componentDidMount() {
            if (this.props.onSetColumns) {
                this.props.onSetColumns(this.props.columns);
            }
        }

        componentDidUpdate( prevProps: Readonly<DataExplorerProps<T>>, prevState: Readonly<{}>, snapshot?: any ): void {
            const { selectedResourceUuid, currentRouteUuid } = this.props;
            if(selectedResourceUuid !== prevProps.selectedResourceUuid || currentRouteUuid !== prevProps.currentRouteUuid) {
                this.setState({
                    msToolbarInDetailsCard: selectedResourceUuid === this.props.currentRouteUuid,
                })
            }
        }

        render() {
            const {
                columns,
                onContextMenu,
                onFiltersChange,
                onSortToggle,
                extractKey,
                rowsPerPage,
                rowsPerPageOptions,
                onColumnToggle,
                searchLabel,
                searchValue,
                onSearch,
                items,
                itemsAvailable,
                onRowClick,
                onRowDoubleClick,
                classes,
                defaultViewIcon,
                defaultViewMessages,
                hideColumnSelector,
                actions,
                paperProps,
                hideSearchInput,
                paperKey,
                fetchMode,
                selectedResourceUuid,
                currentRoute,
                title,
                progressBar,
                doHidePanel,
                doMaximizePanel,
                doUnMaximizePanel,
                panelName,
                panelMaximized,
                elementPath,
                toggleMSToolbar,
                setCheckedListOnStore,
                checkedList,
                working,
            } = this.props;
            return (
                <Paper
                    className={classes.root}
                    {...paperProps}
                    key={paperKey}
                    data-cy={this.props["data-cy"]}
                >
                    <Grid
                        container
                        direction="column"
                        wrap="nowrap"
                        className={classes.container}
                    >
                        <div data-cy="title-wrapper" className={classes.titleWrapper} style={currentRoute?.includes('search-results') || !!progressBar ? {marginBottom: '-20px'} : {}}>
                            {title && (
                                <Grid
                                    item
                                    xs
                                    className={!!progressBar ? classes.subProcessTitle : classes.title}
                                >
                                    {title}
                                </Grid>
                            )}
                            {!!progressBar && progressBar}
                            {this.multiSelectToolbarInTitle && !this.state.msToolbarInDetailsCard && <MultiselectToolbar injectedStyles={classes.msToolbarStyles} />}
                            {(!hideColumnSelector || !hideSearchInput || !!actions) && (
                                <Grid
                                    className={classes.headerMenu}
                                    item
                                    xs
                                >
                                    <Toolbar className={classes.toolbar}>
                                        <Grid container justify="space-between" wrap="nowrap" alignItems="center">
                                            {!hideSearchInput && (
                                                <div className={classes.searchBox}>
                                                    {!hideSearchInput && (
                                                        <SearchInput
                                                            label={searchLabel}
                                                            value={searchValue}
                                                            selfClearProp={""}
                                                            onSearch={onSearch}
                                                        />
                                                    )}
                                                </div>
                                            )}
                                            {actions}
                                            {!hideColumnSelector && (
                                                <ColumnSelector
                                                    columns={columns}
                                                    onColumnToggle={onColumnToggle}
                                                />
                                            )}
                                        </Grid>
                                        {doUnMaximizePanel && panelMaximized && (
                                            <Tooltip
                                                title={`Unmaximize ${panelName || "panel"}`}
                                                disableFocusListener
                                            >
                                                <IconButton onClick={doUnMaximizePanel}>
                                                    <UnMaximizeIcon />
                                                </IconButton>
                                            </Tooltip>
                                        )}
                                        {doMaximizePanel && !panelMaximized && (
                                            <Tooltip
                                                title={`Maximize ${panelName || "panel"}`}
                                                disableFocusListener
                                            >
                                                <IconButton onClick={doMaximizePanel}>
                                                    <MaximizeIcon />
                                                </IconButton>
                                            </Tooltip>
                                        )}
                                        {doHidePanel && (
                                            <Tooltip
                                                title={`Close ${panelName || "panel"}`}
                                                disableFocusListener
                                            >
                                                <IconButton
                                                    disabled={panelMaximized}
                                                    onClick={doHidePanel}
                                                >
                                                    <CloseIcon />
                                                </IconButton>
                                            </Tooltip>
                                        )}
                                    </Toolbar>
                                </Grid>
                            )}
                        </div>
                        {!this.multiSelectToolbarInTitle && <MultiselectToolbar isSubPanel={true} injectedStyles={classes.subpanelToolbarStyles}/>}
                        <Grid
                            item
                            xs="auto"
                            className={classes.dataTable}
                            style={currentRoute?.includes('search-results')  || !!progressBar ? {marginTop: '-10px'} : {}}
                        >
                            <DataTable
                                columns={this.props.contextMenuColumn ? [...columns, this.contextMenuColumn] : columns}
                                items={items}
                                onRowClick={(_, item: T) => onRowClick(item)}
                                onContextMenu={onContextMenu}
                                onRowDoubleClick={(_, item: T) => onRowDoubleClick(item)}
                                onFiltersChange={onFiltersChange}
                                onSortToggle={onSortToggle}
                                extractKey={extractKey}
                                defaultViewIcon={defaultViewIcon}
                                defaultViewMessages={defaultViewMessages}
                                currentRoute={paperKey}
                                toggleMSToolbar={toggleMSToolbar}
                                setCheckedListOnStore={setCheckedListOnStore}
                                checkedList={checkedList}
                                selectedResourceUuid={selectedResourceUuid}
                                setSelectedUuid={this.props.setSelectedUuid}
                                currentRouteUuid={this.props.currentRouteUuid}
                                working={working}
                                isNotFound={this.props.isNotFound}
                            />
                        </Grid>
                        <Grid
                            item
                            xs
                        >
                            <Toolbar className={classes.footer}>
                                {elementPath && (
                                    <Grid container>
                                        <span data-cy="element-path">{elementPath}</span>
                                    </Grid>
                                )}
                                <Grid
                                    container={!elementPath}
                                    justify="flex-end"
                                >
                                    {fetchMode === DataTableFetchMode.PAGINATED ? (
                                        <TablePagination
                                            count={itemsAvailable}
                                            rowsPerPage={rowsPerPage}
                                            rowsPerPageOptions={rowsPerPageOptions}
                                            page={this.props.page}
                                            onChangePage={this.changePage}
                                            onChangeRowsPerPage={this.changeRowsPerPage}
                                            // Disable next button on empty lists since that's not default behavior
                                            nextIconButtonProps={itemsAvailable > 0 ? {} : { disabled: true }}
                                            component="div"
                                        />
                                    ) : (
                                        <Button
                                            variant="text"
                                            size="medium"
                                            onClick={this.loadMore}
                                        >
                                            Load more
                                        </Button>
                                    )}
                                </Grid>
                            </Toolbar>
                        </Grid>
                    </Grid>
                </Paper>
            );
        }

        changePage = (event: React.MouseEvent<HTMLButtonElement>, page: number) => {
            this.props.onChangePage(page);
        };

        changeRowsPerPage: React.ChangeEventHandler<HTMLTextAreaElement | HTMLInputElement> = event => {
            this.props.onChangeRowsPerPage(parseInt(event.target.value, 10));
        };

        loadMore = () => {
            this.props.onLoadMore(this.props.page + 1);
        };

        renderContextMenuTrigger = (item: T) => (
            <Grid
                container
                justify="center"
            >
                <Tooltip
                    title="More options"
                    disableFocusListener
                >
                    <IconButton
                        className={this.props.classes.moreOptionsButton}
                        onClick={event => {
                            event.stopPropagation()
                            this.props.onContextMenu(event, item)
                        }}
                    >
                        <MoreVerticalIcon />
                    </IconButton>
                </Tooltip>
            </Grid>
        );

        contextMenuColumn: DataColumn<any, any> = {
            name: "Actions",
            selected: true,
            configurable: false,
            filters: createTree(),
            key: "context-actions",
            render: this.renderContextMenuTrigger,
        };
    }
);
