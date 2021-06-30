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
import { MoreOptionsIcon } from 'components/icon/icon';
import { PaperProps } from '@material-ui/core/Paper';

type CssRules = 'searchBox' | "toolbar" | "toolbarUnderTitle" | "footer" | "root" | 'moreOptionsButton' | 'title';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    searchBox: {
        paddingBottom: theme.spacing.unit * 2
    },
    toolbar: {
        paddingTop: theme.spacing.unit * 2
    },
    toolbarUnderTitle: {
        paddingTop: 0
    },
    footer: {
        overflow: 'auto'
    },
    root: {
        height: '100%'
    },
    moreOptionsButton: {
        padding: 0
    },
    title: {
        paddingLeft: theme.spacing.unit * 3,
        paddingTop: theme.spacing.unit * 3,
        fontSize: '18px'
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
    hideColumnSelector?: boolean;
    paperProps?: PaperProps;
    actions?: React.ReactNode;
    hideSearchInput?: boolean;
    title?: React.ReactNode;
    paperKey?: string;
    currentItemUuid: string;
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

type DataExplorerProps<T> = DataExplorerDataProps<T> & DataExplorerActionProps<T> & WithStyles<CssRules>;

export const DataExplorer = withStyles(styles)(
    class DataExplorerGeneric<T> extends React.Component<DataExplorerProps<T>> {
        componentDidMount() {
            if (this.props.onSetColumns) {
                this.props.onSetColumns(this.props.columns);
            }
        }
        render() {
            const {
                columns, onContextMenu, onFiltersChange, onSortToggle, working, extractKey,
                rowsPerPage, rowsPerPageOptions, onColumnToggle, searchLabel, searchValue, onSearch,
                items, itemsAvailable, onRowClick, onRowDoubleClick, classes,
                dataTableDefaultView, hideColumnSelector, actions, paperProps, hideSearchInput,
                paperKey, fetchMode, currentItemUuid, title
            } = this.props;
            return <Paper className={classes.root} {...paperProps} key={paperKey}>
                {title && <div className={classes.title}>{title}</div>}
                {(!hideColumnSelector || !hideSearchInput) && <Toolbar className={title ? classes.toolbarUnderTitle : classes.toolbar}>
                    <Grid container justify="space-between" wrap="nowrap" alignItems="center">
                        <div className={classes.searchBox}>
                            {!hideSearchInput && <SearchInput
                                label={searchLabel}
                                value={searchValue}
                                onSearch={onSearch} />}
                        </div>
                        {actions}
                        {!hideColumnSelector && <ColumnSelector
                            columns={columns}
                            onColumnToggle={onColumnToggle} />}
                    </Grid>
                </Toolbar>}
                <DataTable
                    columns={this.props.contextMenuColumn ? [...columns, this.contextMenuColumn] : columns}
                    items={items}
                    onRowClick={(_, item: T) => onRowClick(item)}
                    onContextMenu={onContextMenu}
                    onRowDoubleClick={(_, item: T) => onRowDoubleClick(item)}
                    onFiltersChange={onFiltersChange}
                    onSortToggle={onSortToggle}
                    extractKey={extractKey}
                    working={working}
                    defaultView={dataTableDefaultView}
                    currentItemUuid={currentItemUuid}
                    currentRoute={paperKey} />
                <Toolbar className={classes.footer}>
                    <Grid container justify="flex-end">
                        {fetchMode === DataTableFetchMode.PAGINATED ? <TablePagination
                            count={itemsAvailable}
                            rowsPerPage={rowsPerPage}
                            rowsPerPageOptions={rowsPerPageOptions}
                            page={this.props.page}
                            onChangePage={this.changePage}
                            onChangeRowsPerPage={this.changeRowsPerPage}
                            component="div" /> : <Button
                                variant="text"
                                size="medium"
                                onClick={this.loadMore}
                            >Load more</Button>}
                    </Grid>
                </Toolbar>
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
