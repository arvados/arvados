// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Grid, Paper, Toolbar, StyleRulesCallback, withStyles, WithStyles, TablePagination, IconButton, Tooltip, Button } from '@material-ui/core';
import { ColumnSelector } from "components/column-selector/column-selector";
import { DataTable, DataColumns, DataTableFetchMode } from "components/data-table/data-table";
import { DataColumn } from "components/data-table/data-column";
import { SearchInput } from 'components/search-input/search-input';
import { ArvadosTheme } from "common/custom-theme";
import { createTree } from 'models/tree';
import { DataTableFilters } from 'components/data-table-filters/data-table-filters-tree';
import { CloseIcon, MaximizeIcon, MoreOptionsIcon } from 'components/icon/icon';
import { PaperProps } from '@material-ui/core/Paper';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';

type CssRules = 'searchBox' | 'headerMenu' | "toolbar" | "footer" | "root" | 'moreOptionsButton' | 'title' | 'dataTable' | 'container';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    searchBox: {
        paddingBottom: theme.spacing.unit * 2
    },
    toolbar: {
        paddingTop: theme.spacing.unit,
        paddingRight: theme.spacing.unit * 2,
    },
    footer: {
        overflow: 'auto'
    },
    root: {
        height: '100%',
    },
    moreOptionsButton: {
        padding: 0
    },
    title: {
        display: 'inline-block',
        paddingLeft: theme.spacing.unit * 3,
        paddingTop: theme.spacing.unit * 3,
        fontSize: '18px'
    },
    dataTable: {
        height: '100%',
        overflow: 'auto',
    },
    container: {
        height: '100%',
    },
    headerMenu: {
        float: 'right',
        display: 'inline-block'
    }
});

interface DataExplorerDataProps<T> {
    fetchMode: DataTableFetchMode;
    items: T[];
    itemsAvailable: number;
    columns: DataColumns<T>;
    searchLabel?: string;
    searchValue: string;
    rowsPerPage: number;
    rowsPerPageOptions: number[];
    page: number;
    contextMenuColumn: boolean;
    dataTableDefaultView?: React.ReactNode;
    working?: boolean;
    currentRefresh?: string;
    currentRoute?: string;
    hideColumnSelector?: boolean;
    paperProps?: PaperProps;
    actions?: React.ReactNode;
    hideSearchInput?: boolean;
    title?: React.ReactNode;
    paperKey?: string;
    currentItemUuid: string;
    elementPath?: string;
}

interface DataExplorerActionProps<T> {
    onSetColumns: (columns: DataColumns<T>) => void;
    onSearch: (value: string) => void;
    onRowClick: (item: T) => void;
    onRowDoubleClick: (item: T) => void;
    onColumnToggle: (column: DataColumn<T>) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: T) => void;
    onSortToggle: (column: DataColumn<T>) => void;
    onFiltersChange: (filters: DataTableFilters, column: DataColumn<T>) => void;
    onChangePage: (page: number) => void;
    onChangeRowsPerPage: (rowsPerPage: number) => void;
    onLoadMore: (page: number) => void;
    extractKey?: (item: T) => React.Key;
}

type DataExplorerProps<T> = DataExplorerDataProps<T> &
    DataExplorerActionProps<T> & WithStyles<CssRules> & MPVPanelProps;

export const DataExplorer = withStyles(styles)(
    class DataExplorerGeneric<T> extends React.Component<DataExplorerProps<T>> {
        state = {
            showLoading: false,
            prevRefresh: '',
            prevRoute: '',
        };

        componentDidUpdate(prevProps: DataExplorerProps<T>) {
            const currentRefresh = this.props.currentRefresh || '';
            const currentRoute = this.props.currentRoute || '';

            if (currentRoute !== this.state.prevRoute) {
                // Component already mounted, but the user comes from a route change,
                // like browsing through a project hierarchy.
                this.setState({
                    showLoading: this.props.working,
                    prevRoute: currentRoute,
                });
            }

            if (currentRefresh !== this.state.prevRefresh) {
                // Component already mounted, but the user just clicked the
                // refresh button.
                this.setState({
                    showLoading: this.props.working,
                    prevRefresh: currentRefresh,
                });
            }
            if (this.state.showLoading && !this.props.working) {
                this.setState({
                    showLoading: false,
                });
            }
        }

        componentDidMount() {
            if (this.props.onSetColumns) {
                this.props.onSetColumns(this.props.columns);
            }
            // Component just mounted, so we need to show the loading indicator.
            this.setState({
                showLoading: this.props.working,
                prevRefresh: this.props.currentRefresh || '',
                prevRoute: this.props.currentRoute || '',
            });
        }

        render() {
            const {
                columns, onContextMenu, onFiltersChange, onSortToggle, extractKey,
                rowsPerPage, rowsPerPageOptions, onColumnToggle, searchLabel, searchValue, onSearch,
                items, itemsAvailable, onRowClick, onRowDoubleClick, classes,
                dataTableDefaultView, hideColumnSelector, actions, paperProps, hideSearchInput,
                paperKey, fetchMode, currentItemUuid, title,
                doHidePanel, doMaximizePanel, panelName, panelMaximized, elementPath
            } = this.props;

            return <Paper className={classes.root} {...paperProps} key={paperKey} data-cy={this.props["data-cy"]}>
                <Grid container direction="column" wrap="nowrap" className={classes.container}>
                    <div>
                        {title && <Grid item xs className={classes.title}>{title}</Grid>}
                        {
                            (!hideColumnSelector || !hideSearchInput || !!actions) &&
                            <Grid className={classes.headerMenu} item xs>
                                <Toolbar className={classes.toolbar}>
                                    <Grid container justify="space-between" wrap="nowrap" alignItems="center">
                                        {!hideSearchInput && <div className={classes.searchBox}>
                                            {!hideSearchInput && <SearchInput
                                                label={searchLabel}
                                                value={searchValue}
                                                selfClearProp={currentItemUuid}
                                                onSearch={onSearch} />}
                                        </div>}
                                        {actions}
                                        {!hideColumnSelector && <ColumnSelector
                                            columns={columns}
                                            onColumnToggle={onColumnToggle} />}
                                    </Grid>
                                    { doMaximizePanel && !panelMaximized &&
                                        <Tooltip title={`Maximize ${panelName || 'panel'}`} disableFocusListener>
                                            <IconButton onClick={doMaximizePanel}><MaximizeIcon /></IconButton>
                                        </Tooltip> }
                                    { doHidePanel &&
                                        <Tooltip title={`Close ${panelName || 'panel'}`} disableFocusListener>
                                            <IconButton onClick={doHidePanel}><CloseIcon /></IconButton>
                                        </Tooltip> }
                                </Toolbar>
                            </Grid>
                        }
                    </div>
                <Grid item xs="auto" className={classes.dataTable}><DataTable
                    columns={this.props.contextMenuColumn ? [...columns, this.contextMenuColumn] : columns}
                    items={items}
                    onRowClick={(_, item: T) => onRowClick(item)}
                    onContextMenu={onContextMenu}
                    onRowDoubleClick={(_, item: T) => onRowDoubleClick(item)}
                    onFiltersChange={onFiltersChange}
                    onSortToggle={onSortToggle}
                    extractKey={extractKey}
                    working={this.state.showLoading}
                    defaultView={dataTableDefaultView}
                    currentItemUuid={currentItemUuid}
                    currentRoute={paperKey} /></Grid>
                <Grid item xs><Toolbar className={classes.footer}>
                    {
                        elementPath &&
                        <Grid container>
                            <span data-cy="element-path">
                                {elementPath}
                            </span>
                        </Grid>
                    }
                    <Grid container={!elementPath} justify="flex-end">
                        {fetchMode === DataTableFetchMode.PAGINATED ? <TablePagination
                            count={itemsAvailable}
                            rowsPerPage={rowsPerPage}
                            rowsPerPageOptions={rowsPerPageOptions}
                            page={this.props.page}
                            onChangePage={this.changePage}
                            onChangeRowsPerPage={this.changeRowsPerPage}
                            // Disable next button on empty lists since that's not default behavior
                            nextIconButtonProps={(itemsAvailable > 0) ? {} : {disabled: true}}
                            component="div" /> : <Button
                                variant="text"
                                size="medium"
                                onClick={this.loadMore}
                            >Load more</Button>}
                    </Grid>
                </Toolbar></Grid>
                </Grid>
            </Paper>;
        }

        changePage = (event: React.MouseEvent<HTMLButtonElement>, page: number) => {
            this.props.onChangePage(page);
        }

        changeRowsPerPage: React.ChangeEventHandler<HTMLTextAreaElement | HTMLInputElement> = (event) => {
            this.props.onChangeRowsPerPage(parseInt(event.target.value, 10));
        }

        loadMore = () => {
            this.props.onLoadMore(this.props.page + 1);
        }

        renderContextMenuTrigger = (item: T) =>
            <Grid container justify="center">
                <Tooltip title="More options" disableFocusListener>
                    <IconButton className={this.props.classes.moreOptionsButton} onClick={event => this.props.onContextMenu(event, item)}>
                        <MoreOptionsIcon />
                    </IconButton>
                </Tooltip>
            </Grid>

        contextMenuColumn: DataColumn<any> = {
            name: "Actions",
            selected: true,
            configurable: false,
            filters: createTree(),
            key: "context-actions",
            render: this.renderContextMenuTrigger
        };
    }
);
