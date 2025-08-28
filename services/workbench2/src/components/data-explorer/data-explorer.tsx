// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import {
    Grid,
    Paper,
    Toolbar,
    TablePagination,
    IconButton,
    Tooltip,
    Button,
    Typography,
} from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ColumnSelector } from "components/column-selector/column-selector";
import { DataColumns } from "components/data-table/data-column";
import { DataTable, DataTableFetchMode } from "components/data-table/data-table";
import { DataColumn } from "components/data-table/data-column";
import { SearchInput } from "components/search-input/search-input";
import { ArvadosTheme } from "common/custom-theme";
import { MultiselectToolbar } from "components/multiselect-toolbar/MultiselectToolbar";
import { TCheckedList } from "components/data-table/data-table";
import { createTree } from "models/tree";
import { DataTableFilters } from "components/data-table-filters/data-table-filters";
import { IconType, MoreVerticalIcon } from "components/icon/icon";
import { PaperProps } from "@mui/material/Paper";
import { MPVPanelProps } from "components/multi-panel-view/multi-panel-view";
import classNames from "classnames";
import { InlinePulser } from "components/loading/inline-pulser";
import { isMoreThanOneSelected } from "store/multiselect/multiselect-actions";
import { ProjectResource } from "models/project";
import { Process } from "store/processes/process";
import { ProcessStatusCounts, isAllProcessesPanel, isSharedWithMePanel } from "store/subprocess-panel/subprocess-panel-actions";
import { SUBPROCESS_PANEL_ID, isProcess } from "store/subprocess-panel/subprocess-panel-actions";
import { PROJECT_PANEL_RUN_ID } from "store/project-panel/project-panel-action-bind";
import { ALL_PROCESSES_PANEL_ID } from "store/all-processes-panel/all-processes-panel-action";
import { WORKFLOW_PROCESSES_PANEL_ID } from "store/workflow-panel/workflow-panel-actions";
import { SHARED_WITH_ME_PANEL_ID } from "store/shared-with-me-panel/shared-with-me-panel-actions";
import { ColumnFilterCounts } from "components/data-table-filters/data-table-filters-tree";
import { WorkflowResource } from "models/workflow";

type CssRules =
    | 'titleWrapper'
    | 'searchResultsTitleWrapper'
    | 'msToolbarStyles'
    | 'searchBox'
    | 'headerMenu'
    | 'toolbar'
    | 'footer'
    | 'loadMoreContainer'
    | 'numResults'
    | 'root'
    | 'moreOptionsButton'
    | 'title'
    | 'dataTable'
    | 'container'
    | 'paginationLabel'
    | 'paginationRoot'
    | "subToolbarWrapper"
    | 'runsToolbarWrapper'
    | 'searchResultsToolbar'
    | 'progressWrapper'
    | 'progressWrapperNoTitle';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    titleWrapper: {
        display: "flex",
        justifyContent: "space-between",
        marginTop: "5px",
        marginBottom: "-5px",
    },
    searchResultsTitleWrapper: {
        display: "flex",
        justifyContent: "space-between",
        marginTop: "5px",
        height: "30px",
    },
    msToolbarStyles: {
        marginLeft: "-5px",
    },
    subToolbarWrapper: {
        marginTop: "5px",
        marginLeft: "-15px",
    },
    runsToolbarWrapper: {
        marginTop: "5px",
        marginLeft: "-15px",
    },
    searchResultsToolbar: {
        marginTop: "10px",
        marginBottom: "auto",
    },
    searchBox: {
        paddingBottom: 0,
    },
    toolbar: {
        paddingTop: 0,
        paddingRight: theme.spacing(1),
        paddingLeft: "10px",
    },
    footer: {
        overflow: "auto",
    },
    loadMoreContainer: {
        minWidth: '8rem',
    },
    root: {
        height: "100%",
        flex: 1,
        overflowY: "auto",
        boxShadow: 'none',
    },
    moreOptionsButton: {
        padding: 0,
    },
    numResults: {
        marginTop: 0,
        fontSize: "10px",
        marginLeft: "10px",
        marginBottom: '-0.5rem',
        minWidth: '8.5rem',
    },
    title: {
        display: "inline-block",
        paddingLeft: theme.spacing(2),
        paddingTop: theme.spacing(2),
        fontSize: "18px",
        flexGrow: 1,
        paddingRight: "10px",
    },
    progressWrapper: {
        margin: "14px 0 0",
        paddingLeft: "20px",
        paddingRight: "20px",
    },
    progressWrapperNoTitle: {
        marginTop: '12px',
    },
    dataTable: {
        height: "100%",
        overflowY: "auto",
    },
    container: {
        height: "100%",
    },
    headerMenu: {
        marginLeft: "auto",
        flexBasis: "initial",
        flexGrow: 0,
    },
    paginationLabel: {
        margin: 0,
        padding: 0,
        fontSize: '0.75rem',
    },
    paginationRoot: {
        fontSize: '0.75rem',
        color: theme.palette.grey["600"],
    },
});

interface DataExplorerDataProps<T> {
    id: string;
    fetchMode: DataTableFetchMode;
    items: T[];
    itemsAvailable: number;
    loadingItemsAvailable: boolean;
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
    hideColumnSelector?: boolean;
    paperProps?: PaperProps;
    actions?: React.ReactNode;
    hideSearchInput?: boolean;
    title?: React.ReactNode;
    path?: string;
    currentRouteUuid: string;
    selectedResourceUuid: string;
    elementPath?: string;
    isMSToolbarVisible: boolean;
    checkedList: TCheckedList;
    isNotFound: boolean;
    searchBarValue: string;
    paperClassName?: string;
    forceMultiSelectMode?: boolean;
    detailsPanelResourceUuid: string;
    isDetailsPanelOpen: boolean;
    isSelectedResourceInDataExplorer: boolean;
    parentResource?: ProjectResource | Process | WorkflowResource;
    typeFilter: string;
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
    onPageChange: (page: number) => void;
    onChangeRowsPerPage: (rowsPerPage: number) => void;
    onLoadMore: (page: number) => void;
    extractKey?: (item: T) => React.Key;
    toggleMSToolbar: (isVisible: boolean) => void;
    setCheckedListOnStore: (checkedList: TCheckedList) => void;
    setSelectedUuid: (uuid: string) => void;
    usesDetailsCard: (uuid: string) => boolean;
    loadDetailsPanel: (uuid: string) => void;
    setIsSelectedResourceInDataExplorer: (isIn: boolean) => void;
    fetchProcessStatusCounts: (parentResourceUuid: string, typeFilter?: string) => Promise<ProcessStatusCounts | undefined>;
}

type DataExplorerProps<T> = DataExplorerDataProps<T> & DataExplorerActionProps<T> & WithStyles<CssRules> & MPVPanelProps;

type DataExplorerState = {
    hideToolbar: boolean;
    isSearchResults: boolean;
    columnFilterCounts: ColumnFilterCounts;
};

export enum FilteredColumnNames {
    STATUS = 'Status',
    TYPE = 'Type',
}

export const DataExplorer = withStyles(styles)(
    class DataExplorerGeneric<T> extends React.Component<DataExplorerProps<T>> {
        state: DataExplorerState = {
            hideToolbar: true,
            isSearchResults: false,
            columnFilterCounts: {},
        };

        multiSelectToolbarInTitle = !this.props.title;
        maxItemsAvailable = 0;

        componentDidMount() {
            if (this.props.onSetColumns) {
                this.props.onSetColumns(this.props.columns);
            }
            this.loadFilterCounts();
            this.setState({
                isSearchResults: this.props.path?.includes("search-results") ? true : false ,
            })
        }

        componentDidUpdate( prevProps: Readonly<DataExplorerProps<T>>, prevState: Readonly<DataExplorerState>, snapshot?: any ): void {
            const { selectedResourceUuid, currentRouteUuid, path, usesDetailsCard, setIsSelectedResourceInDataExplorer } = this.props;
            if(selectedResourceUuid !== prevProps.selectedResourceUuid || currentRouteUuid !== prevProps.currentRouteUuid) {
                setIsSelectedResourceInDataExplorer(this.isSelectedResourceInTable(selectedResourceUuid));
                this.setState({
                    hideToolbar: usesDetailsCard(path || '') ? selectedResourceUuid === this.props.currentRouteUuid : false,
                })
            }
            if (this.props.itemsAvailable !== prevProps.itemsAvailable) {
                this.maxItemsAvailable = Math.max(this.maxItemsAvailable, this.props.itemsAvailable);
            }
            if (this.props.searchBarValue !== prevProps.searchBarValue) {
                this.maxItemsAvailable = 0;
            }
            if (this.props.path !== prevProps.path) {
                this.setState({ isSearchResults: this.props.path?.includes("search-results") ? true : false })
            }
            if ((prevProps.items !== this.props.items || this.props.typeFilter !== prevProps.typeFilter)) {
                this.loadFilterCounts();
            }
        }

        loadFilterCounts = () => {
            const { id, columns } = this.props;
            const filterCountColumns = getFilterCountColumns(id, columns);
            const parentUuid = getParentUuid(this.props.parentResource, id);
            filterCountColumns.forEach(columnName => {
                // more columns to fetch for can be added later
                if(columnName === FilteredColumnNames.STATUS) {
                    this.props.fetchProcessStatusCounts(parentUuid, this.props.typeFilter).then(result=>{
                        if(result) {
                            this.setState({
                                columnFilterCounts: {...this.state.columnFilterCounts, [columnName]: result}
                            })
                        }
                    })
                }
            })
        }

        isSelectedResourceInTable = (resourceUuid) => {
            return this.props.items.includes(resourceUuid);
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
                loadingItemsAvailable,
                onRowClick,
                onRowDoubleClick,
                classes,
                defaultViewIcon,
                defaultViewMessages,
                hideColumnSelector,
                actions,
                paperProps,
                hideSearchInput,
                path,
                fetchMode,
                selectedResourceUuid,
                title,
                panelName,
                elementPath,
                toggleMSToolbar,
                setCheckedListOnStore,
                checkedList,
                working,
                paperClassName,
                forceMultiSelectMode,
                detailsPanelResourceUuid,
                loadDetailsPanel,
            } = this.props;
            return (
                <Paper
                    className={classNames(classes.root, paperClassName)}
                    {...paperProps}
                    key={path}
                    data-cy={this.props["data-cy"]}
                    >
                    <Grid
                        container
                        direction="column"
                        wrap="nowrap"
                        className={classes.container}
                >
                    {title && this.state.isSearchResults && (
                        <Grid
                            item
                            xs
                            className={classes.title}
                        >
                            {title}

                        </Grid>
                    )}

                <div data-cy="title-wrapper" className={classNames(this.state.isSearchResults ? classes.searchResultsTitleWrapper : classes.titleWrapper)}>
                    {title && !this.state.isSearchResults && (
                        <Grid
                            item
                            xs
                            className={classes.title}
                        >
                            {title}

                        </Grid>
                    )}
                    {!this.state.hideToolbar
                        && (this.props.isSelectedResourceInDataExplorer || isMoreThanOneSelected(this.props.checkedList))
                        && (this.multiSelectToolbarInTitle
                            ? <MultiselectToolbar injectedStyles={classes.msToolbarStyles} />
                            : <MultiselectToolbar
                                    forceMultiSelectMode={forceMultiSelectMode}
                                    injectedStyles={classNames(panelName === 'Subprocesses' ? classes.subToolbarWrapper : panelName === 'Runs' ? classes.runsToolbarWrapper : '')}/>)
                    }
                    {(!hideColumnSelector || !hideSearchInput || !!actions) && (
                        <Grid
                            className={classes.headerMenu}
                            item
                            xs
                        >
                            <Toolbar className={classes.toolbar}>
                                <Grid container justifyContent="space-between" wrap="nowrap" alignItems="center">
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
                            </Toolbar>
                        </Grid>
                    )}

                </div>
                <Grid
                    item
                    className={classes.dataTable}
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
                        currentRoute={path}
                        toggleMSToolbar={toggleMSToolbar}
                        setCheckedListOnStore={setCheckedListOnStore}
                        checkedList={checkedList}
                        selectedResourceUuid={selectedResourceUuid}
                        setSelectedUuid={this.props.setSelectedUuid}
                        currentRouteUuid={this.props.currentRouteUuid}
                        working={working}
                        isNotFound={this.props.isNotFound}
                        detailsPanelResourceUuid={detailsPanelResourceUuid}
                        loadDetailsPanel={loadDetailsPanel}
                        columnFilterCounts={this.state.columnFilterCounts}
                    />
                </Grid>
                <Grid
                item
                xs
                >
                <Toolbar className={classes.footer}>
                {elementPath && (
                    <Grid container>
                        <span data-cy="element-path">{elementPath.length > 2 ? elementPath : ''}</span>
                    </Grid>
                )}
                <Grid
                container={!elementPath}
                                    justifyContent="flex-end"
                                >
                                    {fetchMode === DataTableFetchMode.PAGINATED ? (
                                        <TablePagination
                                        data-cy="table-pagination"
                                            count={itemsAvailable}
                                            rowsPerPage={rowsPerPage}
                                            rowsPerPageOptions={rowsPerPageOptions}
                                            page={this.props.page}
                                            onPageChange={this.changePage}
                                            onRowsPerPageChange={this.changeRowsPerPage}
                                            labelDisplayedRows={renderPaginationLabel(loadingItemsAvailable)}
                                            nextIconButtonProps={getPaginiationButtonProps(itemsAvailable, loadingItemsAvailable)}
                                            component="div"
                                            classes={{
                                                root: classes.paginationRoot,
                                                selectLabel: classes.paginationLabel,
                                                displayedRows: classes.paginationLabel,
                                            }}
                                        />
                                    ) : (
                                        <Grid className={classes.loadMoreContainer}>
                                            <Typography  className={classes.numResults}>
                                                Showing {items.length} / {this.maxItemsAvailable} results
                                            </Typography>
                                            <Button
                                                size="small"
                                                onClick={this.loadMore}
                                                variant="contained"
                                                color="primary"
                                                style={{width: '100%', margin: '10px'}}
                                                disabled={working || items.length >= itemsAvailable}
                                            >
                                                Load more
                                            </Button>
                                        </Grid>
                                    )}
                                </Grid>
                            </Toolbar>
                        </Grid>
                    </Grid>
                </Paper>
            );
        }

        changePage = (event: React.MouseEvent<HTMLButtonElement>, page: number) => {
            this.props.onPageChange(page);
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
                justifyContent="center"
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
                        size="large">
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

const renderPaginationLabel = (loading: boolean) => ({ from, to, count }) => (
    loading ?
        <InlinePulser/>
        : <>{from}-{to} of {count}</>
);

const getPaginiationButtonProps = (itemsAvailable: number, loading: boolean) => (
    loading
        ? { disabled: false } // Always allow paging while loading total
        : itemsAvailable > 0
            ? { }
            : { disabled: true } // Disable next button on empty lists since that's not default behavior
);

const getFilterCountColumns = (dataExplorerId: string, columns: DataColumns<any, any>) => {
    const goodDataExplorers = [ PROJECT_PANEL_RUN_ID, SUBPROCESS_PANEL_ID, WORKFLOW_PROCESSES_PANEL_ID, ALL_PROCESSES_PANEL_ID, SHARED_WITH_ME_PANEL_ID ];
    const goodColumnNames = [ FilteredColumnNames.STATUS ];
    return columns.reduce((acc: string[], curr) => {
        if(goodDataExplorers.includes(dataExplorerId) && goodColumnNames.includes(curr.name as FilteredColumnNames)) {
            acc.push(curr.name);
        }
        return acc;
    }, [])
};

const getParentUuid = (parentResource: ProjectResource | Process | WorkflowResource | undefined, id: string) => {
    if (parentResource) {
        return isProcess(parentResource)
            ? parentResource.containerRequest.uuid
            : parentResource.uuid
    }
    if (isAllProcessesPanel(id) || isSharedWithMePanel(id)) {
        return id;
    }
    return '';
};